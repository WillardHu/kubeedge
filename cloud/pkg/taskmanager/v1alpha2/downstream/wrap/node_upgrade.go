package wrap

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type NodeUpgradeJobTask struct {
	obj *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
}

// Check that NodeUpgradeJobTask implements the NodeJobTask interface
var _ NodeJobTask = (*NodeUpgradeJobTask)(nil)

func (task NodeUpgradeJobTask) NodeName() string {
	return task.obj.NodeName
}

func (task NodeUpgradeJobTask) CanExecute() bool {
	if task.obj.Action == operationsv1alpha2.NodeUpgradeJobActionInit &&
		task.obj.Status == metav1.ConditionTrue {
		return true
	}
	// For retry situation. The restart of CloudCore will lose the progress in memory,
	// so node tasks that have not obtained the action results need to be retried, and
	// the idempotent processing is handled by EdgeCore.
	return task.obj.Status == ""
}

func (task NodeUpgradeJobTask) Action() (string, error) {
	op := string(task.obj.Action)
	if task.obj.Action == operationsv1alpha2.NodeUpgradeJobActionInit {
		action := actionflow.FlowImagePrePullJob.Find(string(task.obj.Action))
		if action == nil || action.Next(true) == nil {
			return "", fmt.Errorf("no valid action was found")
		}
		op = action.Next(true).Name
	}
	return op, nil
}

func (task NodeUpgradeJobTask) GetObject() any {
	return task.obj
}

type NodeUpgradeJob struct {
	obj *operationsv1alpha2.NodeUpgradeJob
}

// Check that NodeUpgradeJob implements the NodeJob interface
var _ NodeJob = (*NodeUpgradeJob)(nil)

func NewNodeUpgradeJob(obj *operationsv1alpha2.NodeUpgradeJob) *NodeUpgradeJob {
	return &NodeUpgradeJob{obj: obj}
}

func (job NodeUpgradeJob) Name() string {
	return job.obj.Name
}

func (job NodeUpgradeJob) ResourceType() string {
	return operationsv1alpha2.ResourceNodeUpgradeJob
}

func (job NodeUpgradeJob) Concurrency() int {
	return int(job.obj.Spec.Concurrency)
}

func (job NodeUpgradeJob) Spec() any {
	return job.obj.Spec
}

func (job NodeUpgradeJob) Tasks() []NodeJobTask {
	res := make([]NodeJobTask, 0, len(job.obj.Status.NodeStatus))
	for _, it := range job.obj.Status.NodeStatus {
		res = append(res, &NodeUpgradeJobTask{obj: &it})
	}
	return res
}

func (job NodeUpgradeJob) GetObject() any {
	return job.obj
}
