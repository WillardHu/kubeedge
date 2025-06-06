package edgehub

import (
	"sync"
	"time"

	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	msghandler "github.com/kubeedge/kubeedge/edge/pkg/edgehub/messagehandler"
)

// EdgeHub defines edgehub object structure
type EdgeHub struct {
	certManager   certificate.CertManager
	chClient      clients.Adapter
	reconnectChan chan struct{}
	rateLimiter   flowcontrol.RateLimiter
	keeperLock    sync.RWMutex
	enable        bool
}

var _ core.Module = (*EdgeHub)(nil)

var certSync map[string]chan bool

func GetCertSyncChannel() map[string]chan bool {
	return certSync
}

func NewCertSyncChannel() map[string]chan bool {
	certSync = make(map[string]chan bool, 1)
	certSync[modules.EdgeStreamModuleName] = make(chan bool, 1)
	return certSync
}

func newEdgeHub(enable bool) *EdgeHub {
	NewCertSyncChannel()
	return &EdgeHub{
		enable:        enable,
		reconnectChan: make(chan struct{}),
		rateLimiter: flowcontrol.NewTokenBucketRateLimiter(
			float32(config.Config.EdgeHub.MessageQPS),
			int(config.Config.EdgeHub.MessageBurst)),
	}
}

// Register register edgehub
func Register(eh *v1alpha2.EdgeHub, nodeName string) {
	// Initialize the hub configuration
	config.InitConfigure(eh, nodeName)
	// Initialize the message handler
	msghandler.RegisterHandlers()
	// Register self to beehive modules
	core.Register(newEdgeHub(eh.Enable))
}

// Name returns the name of EdgeHub module
func (eh *EdgeHub) Name() string {
	return modules.EdgeHubModuleName
}

// Group returns EdgeHub group
func (eh *EdgeHub) Group() string {
	return modules.HubGroup
}

// Enable indicates whether this module is enabled
func (eh *EdgeHub) Enable() bool {
	return eh.enable
}

// Start sets context and starts the controller
func (eh *EdgeHub) Start() {
	eh.certManager = certificate.NewCertManager(config.Config.EdgeHub, config.Config.NodeName)
	eh.certManager.Start()
	for _, v := range GetCertSyncChannel() {
		v <- true
		close(v)
	}

	go eh.ifRotationDone()

	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub stop")
			return
		default:
		}
		err := eh.initial()
		if err != nil {
			klog.Exitf("failed to init controller: %v", err)
			return
		}

		waitTime := time.Duration(config.Config.Heartbeat) * time.Second * 2

		err = eh.chClient.Init()
		if err != nil {
			klog.Errorf("connection failed: %v, will reconnect after %s", err, waitTime.String())
			time.Sleep(waitTime)
			continue
		}
		// execute hook func after connect
		eh.pubConnectInfo(true)
		go eh.routeToEdge()
		go eh.routeToCloud()
		go eh.keepalive()

		// wait the stop signal
		// stop authinfo manager/websocket connection
		<-eh.reconnectChan
		eh.chClient.UnInit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// sleep one period of heartbeat, then try to connect cloud hub again
		klog.Warningf("connection is broken, will reconnect after %s", waitTime.String())
		time.Sleep(waitTime)

		// clean channel
	clean:
		for {
			select {
			case <-eh.reconnectChan:
			default:
				break clean
			}
		}
	}
}
