package v1alpha2

type JobState string

const (
	JobStateInit       JobState = "Init"
	JobStateInProgress JobState = "InProgress"
	JobStateComplated  JobState = "Complated"
	JobStateFailure    JobState = "Failure"
)

type NodeExecutionState string

const (
	NodeExecutionStateInProgress NodeExecutionState = "InProgress"
	NodeExecutionStateSuccessful NodeExecutionState = "Successful"
	NodeExecutionStateFailure    NodeExecutionState = "Failure"
)

// BasicNodeTaskStatus defines basic fields of node execution status.
// +kubebuilder:validation:Type=object
type BasicNodeTaskStatus struct {
	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`
	// Action represents for the action of the ImagePrePullJob.
	Action NodeUpgradeJobAction `json:"action,omitempty"`
	// Reason represents for the reason of the ImagePrePullJob.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Time represents for the running time of the ImagePrePullJob.
	Time string `json:"time,omitempty"`
}
