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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fsmv1alpha1 "github.com/kubeedge/api/apis/fsm/v1alpha1"
)

const (
	ResourceNodeUpgradeJob = "nodeupgradejob"

	FinalizerNodeUpgradeJob = "kubeedge.io/nodeupgradejob-controller"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeUpgradeJob is used to upgrade edge node from cloud side.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
type NodeUpgradeJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of NodeUpgradeJob.
	// +optional
	Spec NodeUpgradeJobSpec `json:"spec,omitempty"`
	// Most recently observed status of the NodeUpgradeJob.
	// +optional
	Status NodeUpgradeJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeUpgradeJobList is a list of NodeUpgradeJob.
type NodeUpgradeJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of NodeUpgradeJobs.
	Items []NodeUpgradeJob `json:"items"`
}

// NodeUpgradeJobSpec is the specification of the desired behavior of the NodeUpgradeJob.
type NodeUpgradeJobSpec struct {
	// +Required: Version is the EdgeCore version to upgrade.
	Version string `json:"version,omitempty"`

	// TimeoutSeconds limits the duration of the node upgrade job.
	// Default to 300.
	// If set to 0, we'll use the default value 300.
	// +optional
	TimeoutSeconds *uint32 `json:"timeoutSeconds,omitempty"`

	// NodeNames is a request to select some specific nodes. If it is non-empty,
	// the upgrade job simply select these edge nodes to do upgrade operation.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	NodeNames []string `json:"nodeNames,omitempty"`

	// LabelSelector is a filter to select member clusters by labels.
	// It must match a node's labels for the NodeUpgradeJob to be operated on that node.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// Image specifies a container image name, the image contains: keadm and edgecore.
	// keadm is used as upgradetool, to install the new version of edgecore.
	// The image name consists of registry hostname and repository name,
	// if it includes the tag or digest, the tag or digest will be overwritten by Version field above.
	// If the registry hostname is empty, docker.io will be used as default.
	// The default image name is: kubeedge/installation-package.
	// +optional
	Image string `json:"image,omitempty"`

	// ImageDigestGetter define registry v2 interface access configuration.
	// As a transition, it is not required at first, and the image digest is checked when this field is set.
	// +optional
	ImageDigestGetter *ImageDigestGetter `json:"imageDigestGatter"`

	// Concurrency specifies the maximum number of concurrent that edge nodes associated with
	// each CloudCore instance can be upgraded at the same time.
	// The default Concurrency value is 1.
	// +optional
	Concurrency int32 `json:"concurrency,omitempty"`

	// CheckItems specifies the items need to be checked before the task is executed.
	// The default CheckItems value is nil.
	// +optional
	CheckItems []string `json:"checkItems,omitempty"`

	// FailureTolerate specifies the task tolerance failure ratio.
	// The default FailureTolerate value is 0.1.
	// +optional
	FailureTolerate string `json:"failureTolerate,omitempty"`

	// RequireConfirmation specifies whether you need to confirm the upgrade.
	// The default RequireConfirmation value is false.
	// +optional
	RequireConfirmation bool `json:"requireConfirmation,omitempty"`
}

// ImageDigestGetter used to define a method for getting the image digest.
type ImageDigestGetter struct {
	// ARM64 indicates the image digest of the arm64 platform for verification.
	// E.g., sha256:0738039541234567890123456789012345678901234567890123456789012345
	// +optional
	ARM64 string `json:"arm64,omitempty"`

	// AMD64 indicates the image digest of the amd64 platform for verification.
	// E.g., sha256:0738039541234567890123456789012345678901234567890123456789012345
	// +optional
	AMD64 string `json:"amd64,omitempty"`

	// RegistryAPI define registry v2 interface access configuration.
	// Used to automatically gets multiple platform image digests from a remote registry
	// to set values into ARM64 and AMD64 fields.
	// +optional
	RegistryAPI *RegistryAPI `json:"registryAPI,omitempty"`
}

// RegistryAPI used to define registry v2 interface access configuration.
type RegistryAPI struct {
	Host  string `json:"host"`
	Token string `json:"token"`
}

type NodeUpgradeJobAction string

const (
	NodeUpgradeJobActionCheck               NodeUpgradeJobAction = "Check"
	NodeUpgradeJobActionWaitingConfirmation NodeUpgradeJobAction = "WaitingConfirmation"
	NodeUpgradeJobActionBackUp              NodeUpgradeJobAction = "BackUp"
	NodeUpgradeJobActionUpgrade             NodeUpgradeJobAction = "Upgrade"
	NodeUpgradeJobActionRollBack            NodeUpgradeJobAction = "RollBack"
)

// NodeUpgradeJobStatus stores the status of NodeUpgradeJob.
// contains multiple edge nodes upgrade status.
// +kubebuilder:validation:Type=object
type NodeUpgradeJobStatus struct {
	// Phase represents for the phase of the NodeUpgradeJob
	Phase JobPhase `json:"phase"`

	// NodeStatus contains upgrade Status for each edge node.
	NodeStatus []NodeUpgradeJobNodeTaskStatus `json:"nodeStatus,omitempty"`

	// Reason represents for the reason of the NodeUpgradeJob.
	// +optional
	Reason string `json:"reason,omitempty"`

	// State represents for the state phase of the NodeUpgradeJob.
	// There are several possible state values: "", Upgrading, BackingUp, RollingBack and Checking.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	State fsmv1alpha1.State `json:"state,omitempty"`

	// CurrentVersion represents for the current status of the EdgeCore.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	CurrentVersion string `json:"currentVersion,omitempty"`

	// HistoricVersion represents for the historic status of the EdgeCore.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	HistoricVersion string `json:"historicVersion,omitempty"`

	// Time represents for the running time of the ImagePrePullJob.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Time string `json:"time,omitempty"`

	// Event represents for the event of the ImagePrePullJob.
	// There are six possible event values: Init, Check, BackUp, Upgrade, TimeOut, Rollback.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Event string `json:"event,omitempty"`

	// Action represents for the action of the ImagePrePullJob.
	// There are two possible action values: Success, Failure.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Action fsmv1alpha1.Action `json:"action,omitempty"`
}

// NodeUpgradeJobNodeTaskStatus stores the status of Upgrade for each edge node.
// +kubebuilder:validation:Type=object
type NodeUpgradeJobNodeTaskStatus struct {
	// ActionFlow represents for the results of executing the action flow.
	ActionFlow []NodeUpgradeJobActionStatus `json:"actionFlow,omitempty"`

	// CurrentVersion represents for the current status of the EdgeCore.
	CurrentVersion string `json:"currentVersion,omitempty"`

	// HistoricVersion represents for the historic status of the EdgeCore.
	HistoricVersion string `json:"historicVersion,omitempty"`

	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`

	// Phase represents for the phase of the node task.
	Phase NodeTaskPhase `json:"phase,omitempty"`

	// Reason represents the reason for the failure of the node task.
	// +optional
	Reason string `json:"reason,omitempty"`

	// State represents for the upgrade state phase of the edge node.
	// There are several possible state values: "", Upgrading, BackingUp, RollingBack and Checking.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	State fsmv1alpha1.State `json:"state,omitempty"`

	// Event represents for the event of the ImagePrePullJob.
	// There are three possible event values: Init, Check, Pull.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Event string `json:"event,omitempty"`
}

// NodeUpgradeJobActionStatus defines the results of executing the action.
// +kubebuilder:validation:Type=object
type NodeUpgradeJobActionStatus struct {
	// Action represents for the action name
	Action NodeUpgradeJobAction `json:"action,omitempty"`

	// State represents for the status of this image pull on the edge node.
	Status metav1.ConditionStatus `json:"status,omitempty"`

	// Reason represents the reason for the failure of the action.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Time represents for the running time of the node task.
	Time string `json:"time,omitempty"`
}
