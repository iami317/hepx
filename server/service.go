package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"gitee.com/menciis/logx"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fatedier/golib/crypto"
	"github.com/fatedier/golib/net/mux"
	fmux "github.com/hashicorp/yamux"
	quic "github.com/quic-go/quic-go"
	"github.com/samber/lo"

	"github.com/iami317/hepx/pkg/auth"
	v1 "github.com/iami317/hepx/pkg/config/v1"
	modelmetrics "github.com/iami317/hepx/pkg/metrics"
	"github.com/iami317/hepx/pkg/msg"
	"github.com/iami317/hepx/pkg/nathole"
	plugin "github.com/iami317/hepx/pkg/plugin/server"
	"github.com/iami317/hepx/pkg/ssh"
	"github.com/iami317/hepx/pkg/transport"
	httppkg "github.com/iami317/hepx/pkg/util/http"
	netpkg "github.com/iami317/hepx/pkg/util/net"
	"github.com/iami317/hepx/pkg/util/tcpmux"
	"github.com/iami317/hepx/pkg/util/util"
	"github.com/iami317/hepx/pkg/util/vhost"
	"github.com/iami317/hepx/pkg/util/xlog"
	"github.com/iami317/hepx/server/controller"
	"github.com/iami317/hepx/server/group"
	"github.com/iami317/hepx/server/metrics"
	"github.com/iami317/hepx/server/ports"
	"github.com/iami317/hepx/server/proxy"
	"github.com/iami317/hepx/server/visitor"
)

const (
	connReadTimeout       time.Duration = 10 * time.Second
	vhostReadWriteTimeout time.Duration = 30 * time.Second
)

func init() {
	crypto.DefaultSalt = "frp"
	// Disable quic-go's receive buffer warning.
	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")
	// Disable quic-go's ECN support by default. It may cause issues on certain operating systems.
	if os.Getenv("QUIC_GO_DISABLE_ECN") == "" {
		os.Setenv("QUIC_GO_DISABLE_ECN", "true")
	}
}

// Service
type Service struct {
	// Dispatch connections to different handlers listen on same port
	muxer *mux.Mux

	// Accept connections from client
	listener net.Listener

	// Accept connections using kcp
	kcpListener net.Listener

	// Accept connections using quic
	quicListener *quic.Listener

	// Accept connections using websocket
	websocketListener net.Listener

	// Accept frp tls connections
	tlsListener net.Listener

	// Accept pipe connections from ssh tunnel gateway
	sshTunnelListener *netpkg.InternalListener

	// Manage all controllers
	ctlManager *ControlManager

	// Manage all proxies
	pxyManager *proxy.Manager

	// Manage all plugins
	pluginManager *plugin.Manager

	// HTTP vhost router
	httpVhostRouter *vhost.Routers

	// All resource managers and controllers
	rc *controller.ResourceController

	// web server for dashboard UI and apis
	webServer *httppkg.Server

	sshTunnelGateway *ssh.Gateway

	// Verifies authentication based on selected method
	authVerifier auth.Verifier

	tlsConfig *tls.Config

	cfg *v1.ServerConfig

	// service context
	ctx context.Context
	// call cancel to stop service
	cancel context.CancelFunc
	//有登录成功后的其他操作
	OnLoginFn func(ctx context.Context, msg *msg.NewProxy)
}

func NewService(cfg *v1.ServerConfig) (*Service, error) {
	tlsConfig, err := transport.NewServerTLSConfig(
		cfg.Transport.TLS.CertFile,
		cfg.Transport.TLS.KeyFile,
		cfg.Transport.TLS.TrustedCaFile)
	if err != nil {
		return nil, err
	}

	var webServer *httppkg.Server
	if cfg.WebServer.Port > 0 {
		ws, err := httppkg.NewServer(cfg.WebServer)
		if err != nil {
			return nil, err
		}
		webServer = ws

		modelmetrics.EnableMem()
		if cfg.EnablePrometheus {
			modelmetrics.EnablePrometheus()
		}
	}

	svr := &Service{
		ctlManager:    NewControlManager(),
		pxyManager:    proxy.NewManager(),
		pluginManager: plugin.NewManager(),
		rc: &controller.ResourceController{
			VisitorManager: visitor.NewManager(),
			TCPPortManager: ports.NewManager("tcp", cfg.ProxyBindAddr, cfg.AllowPorts),
			UDPPortManager: ports.NewManager("udp", cfg.ProxyBindAddr, cfg.AllowPorts),
		},
		sshTunnelListener: netpkg.NewInternalListener(),
		httpVhostRouter:   vhost.NewRouters(),
		authVerifier:      auth.NewAuthVerifier(cfg.Auth),
		webServer:         webServer,
		tlsConfig:         tlsConfig,
		cfg:               cfg,
		ctx:               context.Background(),
	}
	if webServer != nil {
		webServer.RouteRegister(svr.registerRouteHandlers)
	}

	// Create tcpmux httpconnect multiplexer.
	if cfg.TCPMuxHTTPConnectPort > 0 {
		var l net.Listener
		address := net.JoinHostPort(cfg.ProxyBindAddr, strconv.Itoa(cfg.TCPMuxHTTPConnectPort))
		l, err = net.Listen("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("create server listener error, %v", err)
		}

		svr.rc.TCPMuxHTTPConnectMuxer, err = tcpmux.NewHTTPConnectTCPMuxer(l, cfg.TCPMuxPassthrough, vhostReadWriteTimeout)
		if err != nil {
			return nil, fmt.Errorf("create vhost tcpMuxer error, %v", err)
		}
		logx.Infof("tcpmux httpconnect multiplexer listen on %s, passthough: %v", address, cfg.TCPMuxPassthrough)
	}

	// Init all plugins
	for _, p := range cfg.HTTPPlugins {
		svr.pluginManager.Register(plugin.NewHTTPPluginOptions(p))
		logx.Infof("plugin [%s] has been registered", p.Name)
	}
	svr.rc.PluginManager = svr.pluginManager

	// Init group controller
	svr.rc.TCPGroupCtl = group.NewTCPGroupCtl(svr.rc.TCPPortManager)

	// Init HTTP group controller
	svr.rc.HTTPGroupCtl = group.NewHTTPGroupController(svr.httpVhostRouter)

	// Init TCP mux group controller
	svr.rc.TCPMuxGroupCtl = group.NewTCPMuxGroupCtl(svr.rc.TCPMuxHTTPConnectMuxer)

	// Init 404 not found page
	vhost.NotFoundPagePath = cfg.Custom404Page

	var (
		httpMuxOn  bool
		httpsMuxOn bool
	)
	if cfg.BindAddr == cfg.ProxyBindAddr {
		if cfg.BindPort == cfg.VhostHTTPPort {
			httpMuxOn = true
		}
		if cfg.BindPort == cfg.VhostHTTPSPort {
			httpsMuxOn = true
		}
	}

	// Listen for accepting connections from client.
	address := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.BindPort))
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("create server listener error, %v", err)
	}

	svr.muxer = mux.NewMux(ln)
	svr.muxer.SetKeepAlive(time.Duration(cfg.Transport.TCPKeepAlive) * time.Second)
	go func() {
		_ = svr.muxer.Serve()
	}()
	ln = svr.muxer.DefaultListener()

	svr.listener = ln
	logx.Verbosef("frps tcp listen on %s", address)

	// Listen for accepting connections from client using kcp protocol.
	if cfg.KCPBindPort > 0 {
		address := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.KCPBindPort))
		svr.kcpListener, err = netpkg.ListenKcp(address)
		if err != nil {
			return nil, fmt.Errorf("listen on kcp udp address %s error: %v", address, err)
		}
		logx.Verbosef("frps kcp listen on udp %s", address)
	}

	if cfg.QUICBindPort > 0 {
		address := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.QUICBindPort))
		quicTLSCfg := tlsConfig.Clone()
		quicTLSCfg.NextProtos = []string{"frp"}
		svr.quicListener, err = quic.ListenAddr(address, quicTLSCfg, &quic.Config{
			MaxIdleTimeout:     time.Duration(cfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(cfg.Transport.QUIC.MaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(cfg.Transport.QUIC.KeepalivePeriod) * time.Second,
		})
		if err != nil {
			return nil, fmt.Errorf("listen on quic udp address %s error: %v", address, err)
		}
		logx.Verbosef("frps quic listen on %s", address)
	}

	if cfg.SSHTunnelGateway.BindPort > 0 {
		sshGateway, err := ssh.NewGateway(cfg.SSHTunnelGateway, cfg.ProxyBindAddr, svr.sshTunnelListener)
		if err != nil {
			return nil, fmt.Errorf("create ssh gateway error: %v", err)
		}
		svr.sshTunnelGateway = sshGateway
		logx.Verbosef("frps sshTunnelGateway listen on port %d", cfg.SSHTunnelGateway.BindPort)
	}

	// Listen for accepting connections from client using websocket protocol.
	websocketPrefix := []byte("GET " + netpkg.FrpWebsocketPath)
	websocketLn := svr.muxer.Listen(0, uint32(len(websocketPrefix)), func(data []byte) bool {
		return bytes.Equal(data, websocketPrefix)
	})
	svr.websocketListener = netpkg.NewWebsocketListener(websocketLn)

	// Create http vhost muxer.
	if cfg.VhostHTTPPort > 0 {
		rp := vhost.NewHTTPReverseProxy(vhost.HTTPReverseProxyOptions{
			ResponseHeaderTimeoutS: cfg.VhostHTTPTimeout,
		}, svr.httpVhostRouter)
		svr.rc.HTTPReverseProxy = rp

		address := net.JoinHostPort(cfg.ProxyBindAddr, strconv.Itoa(cfg.VhostHTTPPort))
		server := &http.Server{
			Addr:              address,
			Handler:           rp,
			ReadHeaderTimeout: 60 * time.Second,
		}
		var l net.Listener
		if httpMuxOn {
			l = svr.muxer.ListenHTTP(1)
		} else {
			l, err = net.Listen("tcp", address)
			if err != nil {
				return nil, fmt.Errorf("create vhost http listener error, %v", err)
			}
		}
		go func() {
			_ = server.Serve(l)
		}()
		logx.Verbosef("http service listen on %s", address)
	}

	// Create https vhost muxer.
	if cfg.VhostHTTPSPort > 0 {
		var l net.Listener
		if httpsMuxOn {
			l = svr.muxer.ListenHTTPS(1)
		} else {
			address := net.JoinHostPort(cfg.ProxyBindAddr, strconv.Itoa(cfg.VhostHTTPSPort))
			l, err = net.Listen("tcp", address)
			if err != nil {
				return nil, fmt.Errorf("create server listener error, %v", err)
			}
			logx.Verbosef("https service listen on %s", address)
		}

		svr.rc.VhostHTTPSMuxer, err = vhost.NewHTTPSMuxer(l, vhostReadWriteTimeout)
		if err != nil {
			return nil, fmt.Errorf("create vhost httpsMuxer error, %v", err)
		}
	}

	// frp tls listener
	svr.tlsListener = svr.muxer.Listen(2, 1, func(data []byte) bool {
		// tls first byte can be 0x16 only when vhost https port is not same with bind port
		return int(data[0]) == netpkg.FRPTLSHeadByte || int(data[0]) == 0x16
	})

	// Create nat hole controller.
	nc, err := nathole.NewController(time.Duration(cfg.NatHoleAnalysisDataReserveHours) * time.Hour)
	if err != nil {
		return nil, fmt.Errorf("create nat hole controller error, %v", err)
	}
	svr.rc.NatHoleController = nc
	return svr, nil
}

func (svr *Service) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	svr.ctx = ctx
	svr.cancel = cancel

	// 运行仪表板 Web 服务器。
	if svr.webServer != nil {
		go func() {
			logx.Verbosef("dashboard listen on %s", svr.webServer.Address())
			if err := svr.webServer.Run(); err != nil {
				logx.Warnf("dashboard server exit with error: %v", err)
			}
		}()
	}

	go svr.HandleListener(svr.sshTunnelListener, true)

	if svr.kcpListener != nil {
		go svr.HandleListener(svr.kcpListener, false)
	}
	if svr.quicListener != nil {
		go svr.HandleQUICListener(svr.quicListener)
	}
	go svr.HandleListener(svr.websocketListener, false)
	go svr.HandleListener(svr.tlsListener, false)

	if svr.rc.NatHoleController != nil {
		go svr.rc.NatHoleController.CleanWorker(svr.ctx)
	}

	if svr.sshTunnelGateway != nil {
		go svr.sshTunnelGateway.Run()
	}

	svr.HandleListener(svr.listener, false)

	<-svr.ctx.Done()
	// service context may not be canceled by svr.Close(), we should call it here to release resources
	if svr.listener != nil {
		svr.Close()
	}
}

func (svr *Service) Close() error {
	if svr.kcpListener != nil {
		svr.kcpListener.Close()
		svr.kcpListener = nil
	}
	if svr.quicListener != nil {
		svr.quicListener.Close()
		svr.quicListener = nil
	}
	if svr.websocketListener != nil {
		svr.websocketListener.Close()
		svr.websocketListener = nil
	}
	if svr.tlsListener != nil {
		svr.tlsListener.Close()
		svr.tlsConfig = nil
	}
	if svr.listener != nil {
		svr.listener.Close()
		svr.listener = nil
	}
	svr.ctlManager.Close()
	if svr.cancel != nil {
		svr.cancel()
	}
	return nil
}

func (svr *Service) handleConnection(ctx context.Context, conn net.Conn, internal bool) {
	xl := xlog.FromContextSafe(ctx)

	var (
		rawMsg msg.Message
		err    error
	)

	_ = conn.SetReadDeadline(time.Now().Add(connReadTimeout))
	if rawMsg, err = msg.ReadMsg(conn); err != nil {
		logx.Warnf("Failed to read message: %v", err)
		conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	switch m := rawMsg.(type) {
	case *msg.Login:
		// server plugin hook
		content := &plugin.LoginContent{
			Login:         *m,
			ClientAddress: conn.RemoteAddr().String(),
		}
		retContent, err := svr.pluginManager.Login(content)
		if err == nil {
			m = &retContent.Login
			err = svr.RegisterControl(conn, m, internal)
		}

		// 如果登录失败，请在那里发送错误消息。
		//否则，在控件的工作例程中发送成功消息。
		if err != nil {
			xl.Warnf("register control error: %v", err)
			_ = msg.WriteMsg(conn, &msg.LoginResp{
				Version: "v0.58.1",
				Error:   util.GenerateResponseErrorString("register control error", err, lo.FromPtr(svr.cfg.DetailedErrorsToClient)),
			})
			conn.Close()
		}
	case *msg.NewWorkConn:
		if err := svr.RegisterWorkConn(conn, m); err != nil {
			conn.Close()
		}
	case *msg.NewVisitorConn:
		if err = svr.RegisterVisitorConn(conn, m); err != nil {
			xl.Warnf("register visitor conn error: %v", err)
			_ = msg.WriteMsg(conn, &msg.NewVisitorConnResp{
				ProxyName: m.ProxyName,
				Error:     util.GenerateResponseErrorString("register visitor conn error", err, lo.FromPtr(svr.cfg.DetailedErrorsToClient)),
			})
			conn.Close()
		} else {
			_ = msg.WriteMsg(conn, &msg.NewVisitorConnResp{
				ProxyName: m.ProxyName,
				Error:     "",
			})
		}
	default:
		logx.Warnf("Error message type for the new connection [%s]", conn.RemoteAddr().String())
		conn.Close()
	}
}

// HandleListener accepts connections from client and call handleConnection to handle them.
// If internal is true, it means that this listener is used for internal communication like ssh tunnel gateway.
// TODO(fatedier): Pass some parameters of listener/connection through context to avoid passing too many parameters.
func (svr *Service) HandleListener(l net.Listener, internal bool) {
	// Listen for incoming connections from client.
	for {
		c, err := l.Accept()
		if err != nil {
			logx.Warnf("Listener for incoming connections from client closed")
			return
		}
		// inject xlog object into net.Conn context
		xl := xlog.New()
		ctx := context.Background()

		c = netpkg.NewContextConn(xlog.NewContext(ctx, xl), c)

		if !internal {
			logx.Verbosef("start check TLS connection...")
			originConn := c
			forceTLS := svr.cfg.Transport.TLS.Force
			var isTLS, custom bool
			c, isTLS, custom, err = netpkg.CheckAndEnableTLSServerConnWithTimeout(c, svr.tlsConfig, forceTLS, connReadTimeout)
			if err != nil {
				logx.Warnf("CheckAndEnableTLSServerConnWithTimeout error: %v", err)
				originConn.Close()
				continue
			}
			logx.Verbosef("check TLS connection success, isTLS: %v custom: %v internal: %v", isTLS, custom, internal)
		}

		// Start a new goroutine to handle connection.
		go func(ctx context.Context, frpConn net.Conn) {
			if lo.FromPtr(svr.cfg.Transport.TCPMux) && !internal {
				fmuxCfg := fmux.DefaultConfig()
				fmuxCfg.KeepAliveInterval = time.Duration(svr.cfg.Transport.TCPMuxKeepaliveInterval) * time.Second
				fmuxCfg.LogOutput = io.Discard
				fmuxCfg.MaxStreamWindowSize = 6 * 1024 * 1024
				session, err := fmux.Server(frpConn, fmuxCfg)
				if err != nil {
					logx.Warnf("Failed to create mux connection: %v", err)
					frpConn.Close()
					return
				}

				for {
					stream, err := session.AcceptStream()
					if err != nil {
						logx.Verbosef("Accept new mux stream error: %v", err)
						session.Close()
						return
					}
					go svr.handleConnection(ctx, stream, internal)
				}
			} else {
				svr.handleConnection(ctx, frpConn, internal)
			}
		}(ctx, c)
	}
}

func (svr *Service) HandleQUICListener(l *quic.Listener) {
	// Listen for incoming connections from client.
	for {
		c, err := l.Accept(context.Background())
		if err != nil {
			logx.Warnf("QUICListener for incoming connections from client closed")
			return
		}
		// Start a new goroutine to handle connection.
		go func(ctx context.Context, frpConn quic.Connection) {
			for {
				stream, err := frpConn.AcceptStream(context.Background())
				if err != nil {
					logx.Verbosef("Accept new quic mux stream error: %v", err)
					_ = frpConn.CloseWithError(0, "")
					return
				}
				go svr.handleConnection(ctx, netpkg.QuicStreamToNetConn(stream, frpConn), false)
			}
		}(context.Background(), c)
	}
}

func (svr *Service) RegisterControl(ctlConn net.Conn, loginMsg *msg.Login, internal bool) error {
	// 如果客户端的 RunID 为空，则它是一个新客户端，我们只是创建一个新控制器。
	// 否则，我们会检查是否有一个控制器具有相同的运行 ID。如果是这样，我们释放以前的控制器并启动新的控制器。
	var err error
	if loginMsg.RunID == "" {
		loginMsg.RunID, err = util.RandID()
		if err != nil {
			return err
		}
	}

	ctx := netpkg.NewContextFromConn(ctlConn)
	xl := xlog.FromContextSafe(ctx)
	xl.AppendPrefix(loginMsg.RunID)
	ctx = xlog.NewContext(ctx, xl)
	xl.Tracef("client login info: ip [%s] version [%s] hostname [%s] os [%s] arch [%s]",
		ctlConn.RemoteAddr().String(), loginMsg.Version, loginMsg.Hostname, loginMsg.Os, loginMsg.Arch)

	// Check auth.
	authVerifier := svr.authVerifier
	if internal && loginMsg.ClientSpec.AlwaysAuthPass {
		authVerifier = auth.AlwaysPassVerifier
	}
	if err := authVerifier.VerifyLogin(loginMsg); err != nil {
		return err
	}

	// TODO(fatedier): use SessionContext
	ctl, err := NewControl(ctx, svr.rc, svr.pxyManager, svr.pluginManager, authVerifier, ctlConn, !internal, loginMsg, svr.cfg, svr.OnLoginFn)
	if err != nil {
		xl.Warnf("create new controller error: %v", err)
		// don't return detailed errors to client
		return fmt.Errorf("unexpected error when creating new controller")
	}
	if oldCtl := svr.ctlManager.Add(loginMsg.RunID, ctl); oldCtl != nil {
		oldCtl.WaitClosed()
	}

	ctl.Start()

	// for statistics
	metrics.Server.NewClient()

	go func() {
		// block until control closed
		ctl.WaitClosed()
		svr.ctlManager.Del(loginMsg.RunID, ctl)
	}()
	return nil
}

// RegisterWorkConn 注册一个新的工作连接以控制，代理需要它。
func (svr *Service) RegisterWorkConn(workConn net.Conn, newMsg *msg.NewWorkConn) error {
	xl := netpkg.NewLogFromConn(workConn)
	ctl, exist := svr.ctlManager.GetByID(newMsg.RunID)
	if !exist {
		xl.Warnf("No client control found for run id [%s]", newMsg.RunID)
		return fmt.Errorf("no client control found for run id [%s]", newMsg.RunID)
	}

	// server plugin hook
	content := &plugin.NewWorkConnContent{
		User: plugin.UserInfo{
			User:  ctl.loginMsg.User,
			Metas: ctl.loginMsg.Metas,
			RunID: ctl.loginMsg.RunID,
		},
		NewWorkConn: *newMsg,
	}
	retContent, err := svr.pluginManager.NewWorkConn(content)
	if err == nil {
		newMsg = &retContent.NewWorkConn
		// Check auth.
		err = ctl.authVerifier.VerifyNewWorkConn(newMsg)
	}
	if err != nil {
		xl.Warnf("invalid NewWorkConn with run id [%s]", newMsg.RunID)
		_ = msg.WriteMsg(workConn, &msg.StartWorkConn{
			Error: util.GenerateResponseErrorString("invalid NewWorkConn", err, lo.FromPtr(svr.cfg.DetailedErrorsToClient)),
		})
		return fmt.Errorf("invalid NewWorkConn with run id [%s]", newMsg.RunID)
	}
	return ctl.RegisterWorkConn(workConn)
}

func (svr *Service) RegisterVisitorConn(visitorConn net.Conn, newMsg *msg.NewVisitorConn) error {
	visitorUser := ""
	// TODO(deprecation): Compatible with old versions, can be without runID, user is empty. In later versions, it will be mandatory to include runID.
	// If runID is required, it is not compatible with versions prior to v0.50.0.
	if newMsg.RunID != "" {
		ctl, exist := svr.ctlManager.GetByID(newMsg.RunID)
		if !exist {
			return fmt.Errorf("no client control found for run id [%s]", newMsg.RunID)
		}
		visitorUser = ctl.loginMsg.User
	}
	return svr.rc.VisitorManager.NewConn(newMsg.ProxyName, visitorConn, newMsg.Timestamp, newMsg.SignKey,
		newMsg.UseEncryption, newMsg.UseCompression, visitorUser)
}
