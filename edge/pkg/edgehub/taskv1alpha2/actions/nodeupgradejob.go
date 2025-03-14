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

package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	"github.com/kubeedge/kubeedge/pkg/util/files"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func newNodeUpgradeJobRunner() *ActionRunner {
	logger := klog.Background().WithName("node-upgrade-job-runner")
	config := options.GetEdgeCoreConfig()
	handler := nodeUpgradeJobActionHandler{
		logger: logger,
		backupFiles: []string{
			config.DataBase.DataSource,
			constants.DefaultConfigDir + "edgecore.yaml",
			filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
		},
	}
	runner := &ActionRunner{
		Flow:               actionflow.FlowNodeUpgradeJob,
		ReportActionStatus: handler.reportActionStatus,
		GetSpecSerializer:  handler.getSpecSerializer,
		Logger:             logger,
	}
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionCheck), handler.checkItems)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionWaitingConfirmation), handler.waitingConfirmation)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionBackUp), handler.backup)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionUpgrade), handler.upgrade)
	runner.addAction(string(operationsv1alpha2.NodeUpgradeJobActionRollBack), handler.rollback)
	return runner
}

type nodeUpgradeJobActionResponse struct {
	baseActionResponse
}

// nodeUpgradeJobActionHandler defines action-related functions
type nodeUpgradeJobActionHandler struct {
	backupFiles []string
	logger      logr.Logger
}

func (nodeUpgradeJobActionHandler) checkItems(_ctx context.Context, specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	if err := PreCheck(spec.CheckItems); err != nil {
		resp.err = err
		return resp
	}
	if spec.ImageDigestGatter != nil {
		// TODO: equal image digest ...
	}
	resp.doNext = true
	return resp
}

func (nodeUpgradeJobActionHandler) waitingConfirmation(_ctx context.Context, specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	// If confirmation is required, return false to block the action flow.
	resp.doNext = !spec.RequireConfirmation
	return resp
}

func (h *nodeUpgradeJobActionHandler) backup(_ctx context.Context, _specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	backupPath := filepath.Join(util.KubeEdgeBackupPath, version.Get().String())
	for _, file := range h.backupFiles {
		if err := files.FileCopy(file, filepath.Join(backupPath, filepath.Base(file))); err != nil {
			resp.err = fmt.Errorf("failed to backup file %s, err: %v", file, err)
			return resp
		}
	}
	return resp
}

func (nodeUpgradeJobActionHandler) upgrade(_ctx context.Context, _specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	// TODO: ..
	return resp
}

func (nodeUpgradeJobActionHandler) rollback(_ctx context.Context, _specser SpecSerializer) ActionResponse {
	resp := new(nodeUpgradeJobActionResponse)
	// TODO: ..
	return resp
}

func (nodeUpgradeJobActionHandler) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(d []byte) (any, error) {
		var spec operationsv1alpha2.NodeUpgradeJobSpec
		if err := json.Unmarshal(d, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	})
}

func (nodeUpgradeJobActionHandler) reportActionStatus(jobname, nodename, action string, resp ActionResponse) {
	res := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	body := taskmsg.UpstreamMessage{
		Action: action,
	}
	if err := resp.Error(); err != nil {
		body.Succ = false
		body.Reason = err.Error()
	} else {
		body.Succ = true
	}
	message.ReportNodeTaskStatus(res, body)
}
