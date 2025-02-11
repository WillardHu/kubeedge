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

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func newNodeUpgradeJobRunner() *ActionRunner {
	var funcs nodeUpgradeJobFuncs
	return &ActionRunner{
		Actions: map[string]ActionFun{
			string(operationsv1alpha2.NodeUpgradeJobActionCheck):               funcs.checkItems,
			string(operationsv1alpha2.NodeUpgradeJobActionWaitingConfirmation): funcs.waitingConfirmation,
			string(operationsv1alpha2.NodeUpgradeJobActionConfirm):             funcs.confirm,
		},
		Flow:               actionflow.FlowNodeUpgradeJob,
		ReportActionStatus: funcs.reportActionStatus,
		GetSpecSerializer:  funcs.getSpecSerializer,
	}
}

// nodeUpgradeJobFuncs used to control function scope
type nodeUpgradeJobFuncs struct{}

func (nodeUpgradeJobFuncs) checkItems(_ctx context.Context, specser SpecSerializer) (bool, error) {
	spec, ok := specser.GetSpec().(operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		return false, fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T",
			specser.GetSpec())
	}
	if err := PreCheck(spec.CheckItems); err != nil {
		return false, err
	}
	return true, nil
}

func (nodeUpgradeJobFuncs) waitingConfirmation(_ctx context.Context, specser SpecSerializer) (bool, error) {
	spec, ok := specser.GetSpec().(operationsv1alpha2.NodeUpgradeJobSpec)
	if !ok {
		return false, fmt.Errorf("failed to conv spec to NodeUpgradeJobSpec, actual type %T",
			specser.GetSpec())
	}
	// If confirmation is required, return false to block the action flow.
	return !spec.RequireConfirmation, nil
}

func (nodeUpgradeJobFuncs) confirm(_ctx context.Context, _specser SpecSerializer) (bool, error) {
	// Used to process the confirmation action and transition to backup.
	return true, nil
}

func (nodeUpgradeJobFuncs) backup(_ctx context.Context, _specser SpecSerializer,
) (bool, error) {
	// TODO: ..
	return true, nil
}

func (nodeUpgradeJobFuncs) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(d []byte) (any, error) {
		var spec operationsv1alpha2.NodeUpgradeJobSpec
		if err := json.Unmarshal(d, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	})
}

func (nodeUpgradeJobFuncs) reportActionStatus(jobname, nodename, action string, err error) {
	res := nodetaskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	var errmsg string
	if err != nil {
		errmsg = err.Error()
	}
	body := nodetaskmsg.UpstreamMessage{
		Action: action,
		Succ:   err == nil,
		Reason: errmsg,
	}
	message.ReportNodeTaskStatus(res, body)
}
