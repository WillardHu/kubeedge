package wrap

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type ImagePrePullJobTask struct {
	obj *operationsv1alpha2.ImagePrePullNodeTaskStatus
}

// Check that ImagePrePullJobTask implements the NodeJobTask interface
var _ NodeJobTask = (*ImagePrePullJobTask)(nil)

func (task ImagePrePullJobTask) NodeName() string {
	return task.obj.NodeName
}

func (task ImagePrePullJobTask) CanExecute() bool {
	if task.obj.Action == operationsv1alpha2.ImagePrePullJobActionInit &&
		task.obj.Status == metav1.ConditionTrue {
		return true
	}
	// For retry situation. The restart of CloudCore will lose the progress in memory,
	// so node tasks that have not obtained the action results need to be retried, and
	// the idempotent processing is handled by EdgeCore.
	return task.obj.Status == ""
}

func (task ImagePrePullJobTask) Action() (string, error) {
	op := string(task.obj.Action)
	if task.obj.Action == operationsv1alpha2.ImagePrePullJobActionInit {
		action := actionflow.FlowImagePrePullJob.Find(string(task.obj.Action))
		if action == nil || action.Next(true) == nil {
			return "", fmt.Errorf("no valid action was found")
		}
		op = action.Next(true).Name
	}
	return op, nil
}

func (task ImagePrePullJobTask) GetObject() any {
	return task.obj
}

type ImagePrePullJob struct {
	obj *operationsv1alpha2.ImagePrePullJob
}

// Check that ImagePrePullJob implements the NodeJob interface
var _ NodeJob = (*ImagePrePullJob)(nil)

func NewImagePrepullJob(obj *operationsv1alpha2.ImagePrePullJob) *ImagePrePullJob {
	return &ImagePrePullJob{obj: obj}
}

func (job ImagePrePullJob) Name() string {
	return job.obj.Name
}

func (job ImagePrePullJob) ResourceType() string {
	return operationsv1alpha2.ResourceImagePrePullJob
}

func (job ImagePrePullJob) Concurrency() int {
	return int(job.obj.Spec.ImagePrePullTemplate.Concurrency)
}

func (job ImagePrePullJob) Spec() any {
	return job.obj.Spec
}

func (job ImagePrePullJob) Tasks() []NodeJobTask {
	res := make([]NodeJobTask, 0, len(job.obj.Status.NodeStatus))
	for _, it := range job.obj.Status.NodeStatus {
		res = append(res, &ImagePrePullJobTask{obj: &it})
	}
	return res
}

func (job ImagePrePullJob) GetObject() any {
	return job.obj
}
