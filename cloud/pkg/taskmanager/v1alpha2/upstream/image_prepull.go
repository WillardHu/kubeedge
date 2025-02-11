package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/status"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type ImagePrePullJobHandler struct {
	logger logr.Logger
	crdcli crdcliset.Interface
}

// Check that ImagePrePullJobHandler implements UpstreamHandler interface.
var _ UpstreamHandler = (*ImagePrePullJobHandler)(nil)

// newImagePrePullJobHandler creates a new ImagePrePullJobHandler.
func newImagePrePullJobHandler(ctx context.Context) *ImagePrePullJobHandler {
	logger := klog.FromContext(ctx).
		WithName(fmt.Sprintf("upstream-%s", operationsv1alpha2.ResourceImagePrePullJob))
	return &ImagePrePullJobHandler{
		logger: logger,
		crdcli: client.GetCRDClient(),
	}
}

func (h *ImagePrePullJobHandler) Logger() logr.Logger {
	return h.logger
}

func (h *ImagePrePullJobHandler) FindNodeTaskStatus(ctx context.Context, res taskmsg.Resource,
) (int, any, error) {
	job, err := h.crdcli.OperationsV1alpha2().ImagePrePullJobs().
		Get(ctx, res.JobName, metav1.GetOptions{})
	if err != nil {
		return -1, nil, fmt.Errorf("failed to get image prepull job, err: %v", err)
	}
	idx := -1
	for i, st := range job.Status.NodeStatus {
		if st.NodeName == res.NodeName {
			idx = i
			break
		}
	}
	var nodetask *operationsv1alpha2.ImagePrePullNodeTaskStatus
	if idx >= 0 {
		nodetask = &job.Status.NodeStatus[idx]
	}
	return idx, nodetask, nil
}

func (h *ImagePrePullJobHandler) SetNodeActionStatus(nodetask any, upmsg *taskmsg.UpstreamMessage) error {
	obj, ok := nodetask.(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
	if !ok {
		return fmt.Errorf("failed to convert nodetask to ImagePrePullNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	obj.Action = operationsv1alpha2.ImagePrePullJobAction(upmsg.Action)
	if upmsg.Succ {
		obj.Status = metav1.ConditionTrue
	} else {
		obj.Status = metav1.ConditionFalse
		obj.Reason = upmsg.Reason
	}
	if obj.Action == operationsv1alpha2.ImagePrePullJobActionPull && upmsg.Extend != "" {
		imageMapper := make(map[string]string)
		if err := json.Unmarshal([]byte(upmsg.Extend), &imageMapper); err != nil {
			return fmt.Errorf("failed to unmarshal image mapper, err: %v", err)
		}
		for i := range obj.ImageStatus {
			imgst := &obj.ImageStatus[i]
			if msg, ok := imageMapper[imgst.Image]; ok {
				imgst.State = operationsv1alpha2.NodeExecutionStateFailure
				imgst.Reason = msg
			} else {
				imgst.State = operationsv1alpha2.NodeExecutionStateSuccessful
			}
		}
	}
	return nil
}

func (h *ImagePrePullJobHandler) IsFinalAction(nodetask any) (bool, error) {
	obj, ok := nodetask.(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
	if !ok {
		return false, fmt.Errorf("failed to convert nodetask to ImagePrePullNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	action := actionflow.FlowImagePrePullJob.Find(string(obj.Action))
	if action == nil {
		return false, fmt.Errorf("invalid action %s", obj.Action)
	}
	return action.IsFinal(), nil
}

func (h *ImagePrePullJobHandler) ReleaseExecutorConcurrent(res taskmsg.Resource) error {
	exec, err := executor.GetExecutor(res.ResourceType, res.JobName)
	if err != nil && !errors.Is(err, executor.ErrExecutorNotExists) {
		return fmt.Errorf("failed to get executor, err: %v", err)
	}
	if err := exec.ReleaseOne(); err != nil {
		h.logger.Error(err, "failed to release executor concurrent")
	}
	return nil
}

func (h *ImagePrePullJobHandler) UpdateNodeActionStatus(ctx context.Context,
	jobname string, idx int, nodetask any,
) error {
	obj, ok := nodetask.(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
	if !ok {
		return fmt.Errorf("failed to convert nodetask to ImagePrePullNodeTaskStatus, "+
			"invalid type: %T", nodetask)
	}
	if err := status.UpdateImagePrepullJobNodeTaskStatus(ctx, h.crdcli,
		jobname, idx, obj); err != nil {
		return fmt.Errorf("failed to update node upgrade job status, err: %v", err)
	}
	return nil
}
