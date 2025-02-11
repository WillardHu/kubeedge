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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type fakeFuncs struct {
	stepcount    int
	triggerError bool
}

func (fr *fakeFuncs) step1(_ context.Context, _ SpecSerializer,
) (bool, error) {
	fr.stepcount++
	return true, nil
}

func (fr *fakeFuncs) step2(_ context.Context, _ SpecSerializer,
) (bool, error) {
	fr.stepcount++
	return false, nil
}

func (fr *fakeFuncs) step3(_ context.Context, _ SpecSerializer,
) (bool, error) {
	fr.stepcount++
	return false, errors.New("test error")
}

func (fr *fakeFuncs) step3fail(_ context.Context, _ SpecSerializer,
) (bool, error) {
	fr.stepcount++
	return true, nil
}

func (fr *fakeFuncs) reportActionStatus(_jobname, _nodename, _action string, err error) {
	if err != nil {
		fr.triggerError = true
	}
}

func (fr *fakeFuncs) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(_data []byte) (any, error) {
		return nil, nil
	})
}

func newFakeRunner() (*ActionRunner, *fakeFuncs) {
	funcs := &fakeFuncs{}
	runner := &ActionRunner{
		Actions: map[string]ActionFun{
			"step1":     funcs.step1,
			"step2":     funcs.step2,
			"step3":     funcs.step3,
			"step3fail": funcs.step3fail,
		},
		Flow: &actionflow.Flow{
			First: &actionflow.Action{
				Name: "step1",
				NextSuccessful: &actionflow.Action{
					Name: "step2",
					NextSuccessful: &actionflow.Action{
						Name: "step3",
						NextFailure: &actionflow.Action{
							Name: "step3fail",
						},
					},
				},
			},
		},
		ReportActionStatus: funcs.reportActionStatus,
		GetSpecSerializer:  funcs.getSpecSerializer,
	}
	return runner, funcs
}

func TestRunAction(t *testing.T) {
	jobname, nodename := "test", "node1"
	r, funcs := newFakeRunner()
	r.RunAction(jobname, nodename, "step1", nil)
	require.Equal(t, 2, funcs.stepcount)
	require.False(t, funcs.triggerError)
	r.RunAction(jobname, nodename, "step3", nil)
	require.Equal(t, 3, funcs.stepcount)
	require.True(t, funcs.triggerError)
}
