package test

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func init() {
	core.Register(&stubCloudHub{enable: true})
}

type Attributes struct {
	RoleName  string `json:"iam_role"`
	ProjectID string `json:"project_id"`
}

type stubCloudHub struct {
	wsConn *websocket.Conn
	enable bool
}

var _ core.Module = (*stubCloudHub)(nil)

func (*stubCloudHub) Name() string {
	return "stubCloudHub"
}

func (*stubCloudHub) Group() string {
	//return core.MetaGroup
	return modules.MetaGroup
}

func (tm *stubCloudHub) Enable() bool {
	return tm.enable
}

func (*stubCloudHub) eventReadLoop(conn *websocket.Conn, stop chan bool) {
	for {
		var event interface{}
		err := conn.ReadJSON(&event)
		if err != nil {
			klog.Errorf("read error, connection will be closed: %v", err)
			stop <- true
			return
		}
		klog.Infof("cloud hub receive message %+v", event)
	}
}

func (tm *stubCloudHub) serveEvent(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		klog.Errorf("fail to build websocket connection: %v", err)
		http.Error(w, "fail to upgrade to websocket protocol", http.StatusInternalServerError)
		return
	}
	tm.wsConn = conn
	stop := make(chan bool, 1)
	klog.Info("edge connected")
	go tm.eventReadLoop(conn, stop)
	<-stop
	tm.wsConn = nil
	klog.Info("edge disconnected")
}

func (tm *stubCloudHub) podHandler(w http.ResponseWriter, req *http.Request) {
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			klog.Errorf("read body error %v", err)
			if _, err := w.Write([]byte("read request body error")); err != nil {
				klog.Error(err)
			}
			return
		}
		klog.Infof("request body is %s\n", string(body))

		var pod v1.Pod
		if err = json.Unmarshal(body, &pod); err != nil {
			klog.Errorf("unmarshal request body error %v", err)
			if _, err := w.Write([]byte("unmarshal request body error")); err != nil {
				klog.Error(err)
			}
			return
		}
		var msgReq *model.Message
		switch req.Method {
		case http.MethodPost:
			msgReq = model.NewMessage("").BuildRouter("edgecontroller", "resource",
				"node/fake_node_id/pod/"+string(pod.UID), model.InsertOperation).FillBody(pod)
		case http.MethodDelete:
			msgReq = model.NewMessage("").BuildRouter("edgecontroller", "resource",
				"node/fake_node_id/pod/"+string(pod.UID), model.DeleteOperation).FillBody(pod)
		}

		if tm.wsConn != nil {
			if err := tm.wsConn.WriteJSON(*msgReq); err != nil {
				klog.Error(err)
			}
			klog.Infof("send message to edgehub is %+v\n", *msgReq)
		}

		if _, err := io.WriteString(w, "OK\n"); err != nil {
			klog.Error(err)
		}
	}
}

func (tm *stubCloudHub) Start() {
	defer tm.Cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/{group_id}/events", tm.serveEvent) // for edge-hub
	mux.HandleFunc("/pod", tm.podHandler)               // for pod test
	s := http.Server{
		Addr:    "127.0.0.1:20000",
		Handler: mux,
	}
	klog.Info("Start cloud hub service")
	err := s.ListenAndServe()
	if err != nil {
		klog.Errorf("ListenAndServe: %v", err)
	}
}

func (tm *stubCloudHub) Cleanup() {
	beehiveContext.Cleanup(tm.Name())
}
