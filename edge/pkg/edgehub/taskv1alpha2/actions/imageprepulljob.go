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
	"strings"

	corev1 "k8s.io/api/core/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	klog "k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/image"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func newImagePrePullJobRunner() *ActionRunner {
	var funcs imagePrePullJobFuncs
	return &ActionRunner{
		Actions: map[string]ActionFun{
			string(operationsv1alpha2.ImagePrePullJobActionCheck): funcs.checkItems,
			string(operationsv1alpha2.ImagePrePullJobActionPull):  funcs.pullImages,
		},
		Flow:               actionflow.FlowImagePrePullJob,
		ReportActionStatus: funcs.reportActionStatus,
		GetSpecSerializer:  funcs.getSpecSerializer,
	}
}

type PullError struct {
	errdetail map[string]string
}

func NewPullError(errdetail map[string]string) PullError {
	return PullError{
		errdetail: errdetail,
	}
}

func (perr PullError) Error() string {
	return "there were some failures when pulling images"
}

func (perr PullError) ToJSON() string {
	bff, err := json.Marshal(perr.errdetail)
	if err != nil {
		klog.Errorf("failed to marshal pull error, err: %v", err)
		return ""
	}
	return string(bff)
}

// imagePrePullJobFuncs used to control function scope
type imagePrePullJobFuncs struct{}

func (imagePrePullJobFuncs) checkItems(_ctx context.Context, specser SpecSerializer) (bool, error) {
	spec, ok := specser.GetSpec().(operationsv1alpha2.ImagePrePullJobSpec)
	if !ok {
		return false, fmt.Errorf("failed to conv spec to ImagePrePullJobSpec, actual type %T",
			specser.GetSpec())
	}
	if err := PreCheck(spec.ImagePrePullTemplate.CheckItems); err != nil {
		return false, err
	}
	return true, nil
}

func (imagePrePullJobFuncs) pullImages(ctx context.Context, specser SpecSerializer) (bool, error) {
	spec, ok := specser.GetSpec().(operationsv1alpha2.ImagePrePullJobSpec)
	if !ok {
		return false, fmt.Errorf("failed to conv spec to ImagePrePullJobSpec, actual type %T",
			specser.GetSpec())
	}

	edgecoreCfg := options.GetEdgeCoreConfig()
	imgrt, err := image.NewImageRuntime(
		edgecoreCfg.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		edgecoreCfg.Modules.Edged.TailoredKubeletConfig.RuntimeRequestTimeout.Duration)
	if err != nil {
		return false, err
	}

	var authcfg runtimeapi.AuthConfig
	if imgsec := spec.ImagePrePullTemplate.ImageSecret; imgsec != "" {
		named := strings.Split(imgsec, constants.ResourceSep)
		if len(named) != 2 {
			return false, fmt.Errorf("pull secret format is not correct")
		}
		client := metaclient.New()
		secret, err := client.Secrets(named[0]).Get(named[1])
		if err != nil {
			return false, fmt.Errorf("failed to get secret %s/%s, err: %v", named[0], named[1], err)
		}
		if err = json.Unmarshal(secret.Data[corev1.DockerConfigJsonKey], &authcfg); err != nil {
			return false, fmt.Errorf("failed to unmarshal secret %s/%s to auth config, err: %v",
				named[0], named[1], err)
		}
	}

	errMap := make(map[string]string)
	for _, image := range spec.ImagePrePullTemplate.Images {
		if err := imgrt.PullImage(ctx, image, &authcfg, nil); err != nil {
			errMap[image] = err.Error()
		}
	}
	if len(errMap) > 0 {
		return false, NewPullError(errMap)
	}
	return true, nil
}

func (imagePrePullJobFuncs) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(d []byte) (any, error) {
		var spec operationsv1alpha2.ImagePrePullJobSpec
		if err := json.Unmarshal(d, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	})
}

func (imagePrePullJobFuncs) reportActionStatus(jobname, nodename, action string, err error) {
	res := nodetaskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceImagePrePullJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	var errmsg, extend string
	if err != nil {
		errmsg = err.Error()
		if perr, ok := err.(PullError); ok {
			extend = perr.ToJSON()
		}
	}
	body := nodetaskmsg.UpstreamMessage{
		Action: action,
		Succ:   err == nil,
		Reason: errmsg,
		Extend: extend,
	}
	message.ReportNodeTaskStatus(res, body)
}
