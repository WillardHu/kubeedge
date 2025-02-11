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
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type UpstreamHandler interface {
	// Logger returns the upstream handler logger.
	Logger() logr.Logger

	// FindNodeTaskStatus gets the node task status with the upstream message resource.
	// First query the node job CR by job name, then foreach the array of NodeTaskStatus
	// by node name for find the node task and index value.
	FindNodeTaskStatus(ctx context.Context, res taskmsg.Resource) (int, any, error)

	// SetNodeActionStatus sets the struct of node task status with the upstream message.
	SetNodeActionStatus(nodetask any, upmsg *taskmsg.UpstreamMessage) error

	// IsFinalAction returns true if the node task is the final action.
	IsFinalAction(nodetask any) (bool, error)

	// ReleaseExecutorConcurrent releases the executor concurrent when the node task is the final action.
	ReleaseExecutorConcurrent(res taskmsg.Resource) error

	// UpdateNodeActionStatus updates the status of node action when obtaining upstream message.
	// Parameters idx and nodetask are obtained through FindNodeTaskStatus(..)
	UpdateNodeActionStatus(ctx context.Context, jobname string, idx int, nodetask any) error
}

// upstreamHandlers is the map of upstream handlers.
var upstreamHandlers = make(map[string]UpstreamHandler)

// Init registers the upstream handlers.
func Init(ctx context.Context) {
	upstreamHandlers[operationsv1alpha2.ResourceNodeUpgradeJob] = newNodeUpgradeJobHandler(ctx)
	upstreamHandlers[operationsv1alpha2.ResourceImagePrePullJob] = newImagePrePullJobHandler(ctx)
}

// Start starts the upstream handler.
func Start(ctx context.Context, statusChan <-chan model.Message) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				klog.Info("stop watching upstream messages of node task")
				return
			case msg, ok := <-statusChan:
				if !ok {
					klog.Info("the upstream status channel has been closed")
					return
				}
				data, err := msg.GetContentData()
				if err != nil {
					klog.Warningf("failed to get upstream content data, err: %v", err)
					continue
				}
				var upmsg taskmsg.UpstreamMessage
				if err := json.Unmarshal(data, &upmsg); err != nil {
					klog.Warningf("failed to unmarshal upstream message, err: %v", err)
					continue
				}
				res := taskmsg.ParseResource(msg.GetResource())
				handler, ok := upstreamHandlers[res.ResourceType]
				if !ok {
					klog.Warningf("invalid node task resource type %s", res.ResourceType)
					continue
				}
				if err := updateNodeJobTaskStatus(ctx, res, upmsg, handler); err != nil {
					handler.Logger().Error(err, "failed to update node task status",
						"job name", res.JobName, "node name", res.NodeName)
					continue
				}
			}
		}
	}()
}

// updateNodeJobTaskStatus updates the status of node job task.
func updateNodeJobTaskStatus(ctx context.Context, res taskmsg.Resource,
	upmsg taskmsg.UpstreamMessage, handler UpstreamHandler,
) error {
	idx, nodetask, err := handler.FindNodeTaskStatus(ctx, res)
	if err != nil {
		return fmt.Errorf("failed to find node task status, err: %v", err)
	}
	if idx == -1 {
		handler.Logger().Info("not found node task status", "job name", res.JobName,
			"node name", res.NodeName)
		return nil
	}
	if err := handler.SetNodeActionStatus(nodetask, &upmsg); err != nil {
		return fmt.Errorf("failed to set node task status, err: %v", err)
	}
	final, err := handler.IsFinalAction(nodetask)
	if err != nil {
		// Enough error messages.
		return err
	}
	if final {
		if err := handler.ReleaseExecutorConcurrent(res); err != nil {
			// This error does not affect the process, just logger.
			handler.Logger().Error(err, "failed to release executor concurrent",
				"job name", res.JobName, "node name", res.NodeName)
		}
	}
	if err := handler.UpdateNodeActionStatus(ctx, res.JobName, idx, nodetask); err != nil {
		return fmt.Errorf("failed to update node task status, err: %v", err)
	}
	return nil
}
