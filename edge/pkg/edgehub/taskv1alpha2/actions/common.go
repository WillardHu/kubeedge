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
	"fmt"

	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

// runners is a global map variables,
// used to cache the implementation of the job action runner.
var runners = map[string]*ActionRunner{}

func Init() {
	RegisterRunner(operationsv1alpha2.ResourceNodeUpgradeJob,
		newNodeUpgradeJobRunner())
	RegisterRunner(operationsv1alpha2.ResourceImagePrePullJob,
		newImagePrePullJobRunner())
}

// registerRunner registers the implementation of the job action runner.
func RegisterRunner(name string, runner *ActionRunner) {
	runners[name] = runner
}

// GetRunner returns the implementation of the job action runner.
func GetRunner(name string) *ActionRunner {
	return runners[name]
}

// ActionFun defines the function type of the job action handler.
// The first return value defines whether the action should continue.
// In some scenarios, we want the flow to be paused and continue it
// when triggered elsewhere.
type ActionFun = func(ctx context.Context, specser SpecSerializer) (bool, error)

// baseActionRunner defines the abstruct of the job action runner.
// The implementation of ActionRunner must compose this structure.
type ActionRunner struct {
	// actions defines the function implementation of each action.
	Actions map[string]ActionFun
	// flow defines the action flow of node job.
	Flow *actionflow.Flow
	// ReportActionStatus uses to report status of node action. If the err is not nil,
	// the failure status needs to be reported.
	ReportActionStatus func(jobname, nodename, action string, err error)
	// GetSpecSerializer ... TODO:
	GetSpecSerializer func(specData []byte) (SpecSerializer, error)
}

// Add job action runner to runners.
func (r *ActionRunner) addAction(action string, handler ActionFun) {
	r.Actions[action] = handler
}

// Get job action runner from runners, returns error when not found.
func (r *ActionRunner) mustGetAction(action string) (ActionFun, error) {
	actionFn, ok := r.Actions[action]
	if !ok {
		return nil, fmt.Errorf("invalid job action %s", action)
	}
	return actionFn, nil
}

// RunAction runs the job action.
func (r *ActionRunner) RunAction(jobname, nodename, action string, specData []byte) {
	logr := klog.NewKlogr().WithValues("jobname", jobname)
	ctx := klog.NewContext(context.Background(), logr)
	ser, err := r.GetSpecSerializer(specData)
	if err != nil {
		r.ReportActionStatus(jobname, nodename, action, err)
		return
	}
	for action := r.Flow.Find(action); action != nil && !action.IsFinal(); {
		actionFn, err := r.mustGetAction(action.Name)
		if err != nil {
			r.ReportActionStatus(action.Name, jobname, nodename, err)
			return
		}
		doNext, err := actionFn(ctx, ser)
		r.ReportActionStatus(jobname, nodename, action.Name, err)
		if err != nil {
			action = action.Next(false)
			continue
		}
		if !doNext {
			break
		}
		action = action.Next(true)
	}
}
