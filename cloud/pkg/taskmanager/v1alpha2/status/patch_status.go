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

package status

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/pkg/jsonpath"
)

func UpdateImagePrepullJobNodeTaskStatus(
	ctx context.Context,
	crdcli crdcliset.Interface,
	jobname string,
	idx int,
	obj *operationsv1alpha2.ImagePrePullNodeTaskStatus,
) error {
	jp := jsonpath.New(jsonpath.OpReplace, fmt.Sprintf("/status/nodeStatus/%d", idx))
	if err := jp.SetValue(obj); err != nil {
		return fmt.Errorf("failed to set ImagePrePullNodeTaskStatus to jsonpath value, err: %v", err)
	}
	data, err := jp.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal jsonpath to json data, err: %v", err)
	}
	if _, err := crdcli.OperationsV1alpha2().ImagePrePullJobs().
		Patch(ctx, jobname, types.JSONPatchType, data, metav1.PatchOptions{}); err != nil {
		return err
	}
	return nil
}

func UpdateNodeUpgradeJobNodeTaskStatus(
	ctx context.Context,
	crdcli crdcliset.Interface,
	jobname string,
	idx int,
	obj *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus,
) error {
	jp := jsonpath.New(jsonpath.OpReplace, fmt.Sprintf("/status/nodeStatus/%d", idx))
	if err := jp.SetValue(obj); err != nil {
		return fmt.Errorf("failed to set NodeUpgradeJobNodeTaskStatus to jsonpath value, err: %v", err)
	}
	data, err := jp.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal jsonpath to json data, err: %v", err)
	}
	if _, err := crdcli.OperationsV1alpha2().NodeUpgradeJobs().
		Patch(ctx, jobname, types.JSONPatchType, data, metav1.PatchOptions{}); err != nil {
		return err
	}
	return nil
}
