/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/downstream/wrap"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	"github.com/kubeedge/kubeedge/pkg/util/slices"
)

var (
	nodeTaskExecutors sync.Map

	ErrExecutorNotExists = errors.New("executor not exists")
)

func executorsKey(resourceType, jobName string) string {
	return strings.Join([]string{resourceType, jobName}, "/")
}

func NewNodeTaskExecutor(ctx context.Context, job wrap.NodeJob,
) (*NodeTaskExecutor, bool, error) {
	key := executorsKey(job.ResourceType(), job.Name())
	actual, loaded := nodeTaskExecutors.LoadOrStore(key, &NodeTaskExecutor{
		job:  job,
		pool: make(chan struct{}, job.Concurrency()),
	})
	executor, ok := actual.(*NodeTaskExecutor)
	if !ok {
		return nil, false,
			fmt.Errorf("failed to convert %s executor to NodeTaskExecutor, actual %T",
				key, executor)
	}
	return executor, loaded, nil
}

func GetExecutor(resourceType, jobName string) (*NodeTaskExecutor, error) {
	key := executorsKey(resourceType, jobName)
	actual, loaded := nodeTaskExecutors.Load(key)
	if !loaded {
		return nil, ErrExecutorNotExists
	}
	executor, ok := actual.(*NodeTaskExecutor)
	if !ok {
		return nil, fmt.Errorf("failed to convert %s executor to NodeTaskExecutor, actual %T",
			key, executor)
	}
	return executor, nil
}

type NodeTaskExecutor struct {
	job          wrap.NodeJob
	pool         chan struct{}
	interrupted  atomic.Bool
	messageLayer messagelayer.MessageLayer
}

func (executor *NodeTaskExecutor) Execute(
	ctx context.Context,
	connectedNodes []string,
	handleErr func(ctx context.Context, job wrap.NodeJob, errTask wrap.NodeJobTask, err error),
) {
	defer nodeTaskExecutors.Delete(executor.job.Name())

	logger := klog.FromContext(ctx).WithName("executor").WithValues("jobname", executor.job.Name())
	tasks := executor.job.Tasks()
	for i := range tasks {
		if executor.interrupted.Load() {
			return
		}
		task := tasks[i]
		if !slices.In(connectedNodes, task.NodeName()) {
			// TODO: 考虑重试？因为CloudCore 重启后如果有任务，会先触发任务后再与边端节点产生连接
			// 这会导致任务丢失
			logger.Info("the node has no connection to the current cloudcore instance, skip it",
				"nodename", task.NodeName())
			continue
		}
		if !task.CanExecute() {
			logger.Info("the node does not meet the execution conditions, skip it",
				"nodename", task.NodeName())
			continue
		}
		executor.pool <- struct{}{}

		msgres := taskmsg.Resource{
			APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
			ResourceType: executor.job.ResourceType(),
			JobName:      executor.job.Name(),
			NodeName:     task.NodeName(),
		}
		action, err := task.Action()
		if err != nil {
			handleErr(ctx, executor.job, task, fmt.Errorf("failed to get node task action, err: %v", err))
			return
		}
		msg := messagelayer.BuildNodeTaskRouter(msgres, action).
			FillBody(executor.job.Spec())
		if err := executor.messageLayer.Send(*msg); err != nil {
			handleErr(ctx, executor.job, task, fmt.Errorf("failed to send message to edge, err: %v", err))
			return
		}
	}
}

// ReleaseOne releases a concurrent resource
func (executor *NodeTaskExecutor) ReleaseOne() error {
	select {
	case <-executor.pool:
	default:
		return errors.New("no tasks are running")
	}
	return nil
}

func (executor *NodeTaskExecutor) Interrupt() {
	executor.interrupted.Store(true)
}
