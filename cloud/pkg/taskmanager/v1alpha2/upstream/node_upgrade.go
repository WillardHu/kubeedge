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

package upstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/status"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type NodeUpgradeJobHandler struct {
	logger logr.Logger
	crdcli crdcliset.Interface
}

// Check that NodeUpgradeJobHandler implements UpstreamHandler interface.
var _ UpstreamHandler = (*NodeUpgradeJobHandler)(nil)

// newNodeUpgradeJobHandler creates a new NodeUpgradeJobHandler.
func newNodeUpgradeJobHandler(ctx context.Context) *NodeUpgradeJobHandler {
	logger := klog.FromContext(ctx).
		WithName(fmt.Sprintf("upstream-%s", operationsv1alpha2.ResourceNodeUpgradeJob))
	return &NodeUpgradeJobHandler{
		logger: logger,
		crdcli: client.GetCRDClient(),
	}
}

func (h *NodeUpgradeJobHandler) Logger() logr.Logger {
	return h.logger
}

func (h *NodeUpgradeJobHandler) FindNodeTaskStatus(ctx context.Context, res taskmsg.Resource,
) (int, any, error) {
	job, err := h.crdcli.OperationsV1alpha2().NodeUpgradeJobs().
		Get(ctx, res.JobName, metav1.GetOptions{})
	if err != nil {
		return -1, nil, fmt.Errorf("failed to get node upgrade job, err: %v", err)
	}
	idx := -1
	for i, st := range job.Status.NodeStatus {
		if st.NodeName == res.NodeName {
			idx = i
			break
		}
	}
	var nodetask *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
	if idx >= 0 {
		nodetask = &job.Status.NodeStatus[idx]
	}
	return idx, nodetask, nil
}

func (h *NodeUpgradeJobHandler) SetNodeActionStatus(nodetask any, upmsg *taskmsg.UpstreamMessage) error {
	obj, ok := nodetask.(*operationsv1alpha2.NodeUpgradeJobNodeTaskStatus)
	if !ok {
		return fmt.Errorf("failed to convert nodetask to NodeUpgradeJobNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	obj.Action = operationsv1alpha2.NodeUpgradeJobAction(upmsg.Action)
	if upmsg.Succ {
		obj.Status = metav1.ConditionTrue
	} else {
		obj.Status = metav1.ConditionFalse
		obj.Reason = upmsg.Reason
	}
	// TODO: handle version
	return nil
}

func (h *NodeUpgradeJobHandler) IsFinalAction(nodetask any) (bool, error) {
	obj, ok := nodetask.(*operationsv1alpha2.NodeUpgradeJobNodeTaskStatus)
	if !ok {
		return false, fmt.Errorf("failed to convert nodetask to NodeUpgradeJobNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	action := actionflow.FlowNodeUpgradeJob.Find(string(obj.Action))
	if action == nil {
		return false, fmt.Errorf("invalid action %s", obj.Action)
	}
	return action.IsFinal(), nil
}

func (h *NodeUpgradeJobHandler) ReleaseExecutorConcurrent(res taskmsg.Resource) error {
	exec, err := executor.GetExecutor(res.ResourceType, res.JobName)
	if err != nil && !errors.Is(err, executor.ErrExecutorNotExists) {
		return fmt.Errorf("failed to get executor, err: %v", err)
	}
	if err := exec.ReleaseOne(); err != nil {
		h.logger.Error(err, "failed to release executor concurrent")
	}
	return nil
}

func (h *NodeUpgradeJobHandler) UpdateNodeActionStatus(ctx context.Context,
	jobname string, idx int, nodetask any,
) error {
	obj, ok := nodetask.(*operationsv1alpha2.NodeUpgradeJobNodeTaskStatus)
	if !ok {
		return fmt.Errorf("failed to convert nodetask to NodeUpgradeJobNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	if err := status.UpdateNodeUpgradeJobNodeTaskStatus(ctx, h.crdcli,
		jobname, idx, obj); err != nil {
		return fmt.Errorf("failed to update node upgrade job status, err: %v", err)
	}
	return nil
}
