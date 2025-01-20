package client

import (
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// NodesGetter to get node interface
type NodesGetter interface {
	Nodes(namespace string) NodesInterface
}

// NodesInterface is interface for client nodes
type NodesInterface interface {
	Create(*corev1.Node) (*corev1.Node, error)
	Update(*corev1.Node) error
	Patch(name string, patchBytes []byte) (*corev1.Node, error)
	Delete(name string) error
	Get(name string) (*corev1.Node, error)
}

type nodes struct {
	namespace string
	send      SendInterface
}

// NodeResp represents node response from the api-server
type NodeResp struct {
	Object *corev1.Node
	Err    apierrors.StatusError
}

func newNodes(namespace string, s SendInterface) *nodes {
	return &nodes{
		send:      s,
		namespace: namespace,
	}
}

func (c *nodes) Create(node *corev1.Node) (*corev1.Node, error) {
	if node.Kind == "" {
		node.Kind = reflect.TypeOf(corev1.Node{}).Name()
	}
	if node.APIVersion == "" {
		node.APIVersion = corev1.SchemeGroupVersion.String()
	}
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNode, node.Name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, node)
	resp, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("create node failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}

	return handleNodeResp(content)
}

func (c *nodes) Update(node *corev1.Node) error {
	if node.Kind == "" {
		node.Kind = reflect.TypeOf(corev1.Node{}).Name()
	}
	if node.APIVersion == "" {
		node.APIVersion = corev1.SchemeGroupVersion.String()
	}
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNode, node.Name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, node)
	_, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return fmt.Errorf("update node failed, err: %v", err)
	}
	return nil
}

func (c *nodes) Patch(name string, data []byte) (*corev1.Node, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNodePatch, name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, string(data))
	resp, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("update node failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}

	return handleNodeResp(content)
}

func (c *nodes) Delete(string) error {
	return nil
}

func (c *nodes) Get(name string) (*corev1.Node, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNode, name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("get node failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == modules.MetaManagerModuleName {
		return handleNodeFromMetaDB(name, content)
	}
	return handleNodeFromMetaManager(content)
}

func handleNodeFromMetaDB(name string, content []byte) (*corev1.Node, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node list from db failed, err: %v", err)
	}
	size := len(lists)
	if size == 0 {
		klog.Warningf("not found node '%s' from meta db", name)
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    corev1.GroupName,
			Resource: reflect.TypeOf(corev1.Node{}).Name(),
		}, name)
	}
	if size > 1 {
		return nil, fmt.Errorf("node length from meta db is %d", len(lists))
	}

	var node corev1.Node
	err = json.Unmarshal([]byte(lists[0]), &node)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node from db failed, err: %v", err)
	}
	return &node, nil
}

func handleNodeFromMetaManager(content []byte) (*corev1.Node, error) {
	var node corev1.Node
	err := json.Unmarshal(content, &node)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node failed, err: %v", err)
	}
	return &node, nil
}

func handleNodeResp(content []byte) (*corev1.Node, error) {
	var nodeResp NodeResp
	err := json.Unmarshal(content, &nodeResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node failed, err: %v", err)
	}

	if reflect.DeepEqual(nodeResp.Err, apierrors.StatusError{}) {
		return nodeResp.Object, nil
	}
	return nodeResp.Object, &nodeResp.Err
}
