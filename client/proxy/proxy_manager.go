package proxy

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/samber/lo"

	"github.com/iami317/hepx/client/event"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	"github.com/iami317/hepx/pkg/msg"
	"github.com/iami317/hepx/pkg/transport"
	"github.com/iami317/hepx/pkg/util/xlog"
)

type Manager struct {
	proxies            map[string]*Wrapper
	msgTransporter     transport.MessageTransporter
	inWorkConnCallback func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool

	closed bool
	mu     sync.RWMutex

	clientCfg *v1.ClientCommonConfig

	ctx context.Context
}

func NewManager(
	ctx context.Context,
	clientCfg *v1.ClientCommonConfig,
	msgTransporter transport.MessageTransporter,
) *Manager {
	return &Manager{
		proxies:        make(map[string]*Wrapper),
		msgTransporter: msgTransporter,
		closed:         false,
		clientCfg:      clientCfg,
		ctx:            ctx,
	}
}

func (pm *Manager) StartProxy(name string, remoteAddr string, serverRespErr string) error {
	pm.mu.RLock()
	pxy, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("proxy [%s] not found", name)
	}

	err := pxy.SetRunningStatus(remoteAddr, serverRespErr)
	if err != nil {
		return err
	}
	return nil
}

func (pm *Manager) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	pm.inWorkConnCallback = cb
}

func (pm *Manager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, pxy := range pm.proxies {
		pxy.Stop()
	}
	pm.proxies = make(map[string]*Wrapper)
}

func (pm *Manager) HandleWorkConn(name string, workConn net.Conn, m *msg.StartWorkConn) {
	pm.mu.RLock()
	pw, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if ok {
		pw.InWorkConn(workConn, m)
	} else {
		workConn.Close()
	}
}

func (pm *Manager) HandleEvent(payload interface{}) error {
	var m msg.Message
	switch e := payload.(type) {
	case *event.StartProxyPayload:
		m = e.NewProxyMsg
	case *event.CloseProxyPayload:
		m = e.CloseProxyMsg
	default:
		return event.ErrPayloadType
	}

	return pm.msgTransporter.Send(m)
}

func (pm *Manager) GetAllProxyStatus() []*WorkingStatus {
	ps := make([]*WorkingStatus, 0)
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		ps = append(ps, pxy.GetStatus())
	}
	return ps
}

func (pm *Manager) GetProxyStatus(name string) (*WorkingStatus, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pxy, ok := pm.proxies[name]; ok {
		return pxy.GetStatus(), true
	}
	return nil, false
}

func (pm *Manager) UpdateAll(proxyCfgs []v1.ProxyConfigurer) {
	xl := xlog.FromContextSafe(pm.ctx)
	proxyCfgsMap := lo.KeyBy(proxyCfgs, func(c v1.ProxyConfigurer) string {
		return c.GetBaseConfig().Name
	})
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delPxyNames := make([]string, 0)
	for name, pxy := range pm.proxies {
		del := false
		cfg, ok := proxyCfgsMap[name]
		if !ok || !reflect.DeepEqual(pxy.Cfg, cfg) {
			del = true
		}

		if del {
			delPxyNames = append(delPxyNames, name)
			delete(pm.proxies, name)
			pxy.Stop()
		}
	}
	if len(delPxyNames) > 0 {
		xl.Infof("proxy removed: %s", delPxyNames)
	}

	addPxyNames := make([]string, 0)
	for _, cfg := range proxyCfgs {
		name := cfg.GetBaseConfig().Name
		if _, ok := pm.proxies[name]; !ok {
			pxy := NewWrapper(pm.ctx, cfg, pm.clientCfg, pm.HandleEvent, pm.msgTransporter)
			if pm.inWorkConnCallback != nil {
				pxy.SetInWorkConnCallback(pm.inWorkConnCallback)
			}
			pm.proxies[name] = pxy
			addPxyNames = append(addPxyNames, name)

			pxy.Start()
		}
	}
	if len(addPxyNames) > 0 {
		xl.Tracef("proxy added: %s", addPxyNames)
	}
}
