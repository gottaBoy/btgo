package bnet

import (
	"btgo/bconf"
	"btgo/bdecoder"
	"btgo/biface"
	"btgo/blogger"
	"btgo/bpack"
	"btgo/logo"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xtaci/kcp-go"
)

// Server interface implementation, defines a Server service class
// (接口实现，定义一个Server服务类)
type Server struct {
	// Name of the server (服务器的名称)
	Name string
	//tcp4 or other
	Network string
	// IP version (e.g. "tcp4") - 服务绑定的IP地址
	IP string
	// IP address the server is bound to (服务绑定的端口)
	Port int
	// 服务绑定的websocket 端口 (Websocket port the server is bound to)
	WsPort int
	// 服务绑定的kcp 端口 (kcp port the server is bound to)
	KcpPort int

	// Current server's message handler module, used to bind MsgID to corresponding processing methods
	// (当前Server的消息管理模块，用来绑定MsgID和对应的处理方法)
	msgHandler biface.IMsgHandler

	// Routing mode (路由模式)
	RouterSlicesMode bool

	// Current server's connection manager (当前Server的链接管理器)
	ConnMgr biface.IConnManager

	// Hook function called when a new connection is established
	// (该Server的连接创建时Hook函数)
	onConnStart func(conn biface.IConnection)

	// Hook function called when a connection is terminated
	// (该Server的连接断开时的Hook函数)
	onConnStop func(conn biface.IConnection)

	// Data packet encapsulation method
	// (数据报文封包方式)
	packet biface.IDataPack

	// Asynchronous capture of connection closing status
	// (异步捕获链接关闭状态)
	exitChan chan struct{}

	// Decoder for dealing with message fragmentation and reassembly
	// (断粘包解码器)
	decoder biface.IDecoder

	// Heartbeat checker
	// (心跳检测器)
	hc biface.IHeartbeatChecker

	// websocket
	upgrader *websocket.Upgrader

	// websocket connection authentication
	websocketAuth func(r *http.Request) error

	// connection id
	connId uint64
}

// newServerWithConfig creates a server handle based on config
// (根据config创建一个服务器句柄)
func newServerWithConfig(config *bconf.Config, network string, opts ...Option) biface.IServer {
	logo.PrintLogo()

	s := &Server{
		Name:             config.Name,
		Network:          network,
		IP:               config.Host,
		Port:             config.TCPPort,
		WsPort:           config.WsPort,
		KcpPort:          config.KcpPort,
		msgHandler:       NewMsgHandler(),
		RouterSlicesMode: config.RouterSlicesMode,
		ConnMgr:          newConnManager(),
		exitChan:         nil,
		// Default to using btgo's TLV data pack format
		// (默认使用btgo的TLV封包方式)
		packet:  bpack.Factory().NewPack(biface.BtgoDataPack),
		decoder: bdecoder.NewTLVDecoder(), // Default to using TLV decode (默认使用TLV的解码方式)
		upgrader: &websocket.Upgrader{
			ReadBufferSize: int(config.IOReadBuffSize),
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Display current configuration information
	// (提示当前配置信息)
	config.Show()

	return s
}

// NewServer creates a server handle
// (创建一个服务器句柄)
func NewServer(opts ...Option) biface.IServer {
	return newServerWithConfig(bconf.GlobalObject, "tcp", opts...)
}

// NewUserConfServer creates a server handle using user-defined configuration
// (创建一个服务器句柄)
func NewUserConfServer(config *bconf.Config, opts ...Option) biface.IServer {

	// Refresh user configuration to global configuration variable
	// (刷新用户配置到全局配置变量)
	bconf.UserConfToGlobal(config)

	s := newServerWithConfig(config, "tcp4", opts...)
	return s
}

// NewDefaultRouterSlicesServer creates a server handle with a default RouterRecovery processor.
// (创建一个默认自带一个Recover处理器的服务器句柄)
func NewDefaultRouterSlicesServer(opts ...Option) biface.IServer {
	bconf.GlobalObject.RouterSlicesMode = true
	s := newServerWithConfig(bconf.GlobalObject, "tcp", opts...)
	s.Use(RouterRecovery)
	return s
}

// NewUserConfDefaultRouterSlicesServer creates a server handle with user-configured options and a default Recover handler.
// If the user does not wish to use the Use method, they should use NewUserConfServer instead.
// (创建一个用户配置的自带一个Recover处理器的服务器句柄，如果用户不希望Use这个方法，那么应该使用NewUserConfServer)
func NewUserConfDefaultRouterSlicesServer(config *bconf.Config, opts ...Option) biface.IServer {

	if !config.RouterSlicesMode {
		panic("RouterSlicesMode is false")
	}

	// Refresh user configuration to global configuration variable (刷新用户配置到全局配置变量)
	bconf.UserConfToGlobal(config)

	s := newServerWithConfig(bconf.GlobalObject, "tcp4", opts...)
	s.Use(RouterRecovery)
	return s
}

func (s *Server) StartConn(conn biface.IConnection) {
	// HeartBeat check
	if s.hc != nil {
		// Clone a heart-beat checker from the server side
		heartBeatChecker := s.hc.Clone()

		// Bind current connection
		heartBeatChecker.BindConn(conn)
	}

	// Start processing business for the current connection
	conn.Start()
}

func (s *Server) ListenTcpConn() {
	// 1. Get a TCP address
	addr, err := net.ResolveTCPAddr(s.Network, fmt.Sprintf("%s:%d", s.IP, s.Port))
	if err != nil {
		blogger.Ins().ErrorF("[START] resolve tcp addr err: %v\n", err)
		return
	}

	// 2. Listen to the server address
	var listener net.Listener
	if bconf.GlobalObject.CertFile != "" && bconf.GlobalObject.PrivateKeyFile != "" {
		// Read certificate and private key
		crt, err := tls.LoadX509KeyPair(bconf.GlobalObject.CertFile, bconf.GlobalObject.PrivateKeyFile)
		if err != nil {
			panic(err)
		}

		// TLS connection
		tlsConfig := &tls.Config{}
		tlsConfig.Certificates = []tls.Certificate{crt}
		tlsConfig.Time = time.Now
		tlsConfig.Rand = rand.Reader
		listener, err = tls.Listen(s.Network, fmt.Sprintf("%s:%d", s.IP, s.Port), tlsConfig)
		if err != nil {
			panic(err)
		}
	} else {
		listener, err = net.ListenTCP(s.Network, addr)
		if err != nil {
			panic(err)
		}
	}

	// 3. Start server network connection business
	go func() {
		for {
			// 3.1 Set the maximum connection control for the server. If it exceeds the maximum connection, wait.
			// (设置服务器最大连接控制,如果超过最大连接，则等待)
			if s.ConnMgr.Len() >= bconf.GlobalObject.MaxConnSize {
				blogger.Ins().InfoF("Exceeded the maxConnNum:%d, Wait:%d", bconf.GlobalObject.MaxConnSize, AcceptDelay.duration)
				AcceptDelay.Delay()
				continue
			}
			// 3.2 Block and wait for a client to establish a connection request.
			// (阻塞等待客户端建立连接请求)
			conn, err := listener.Accept()
			if err != nil {
				//Go 1.17+
				if errors.Is(err, net.ErrClosed) {
					blogger.Ins().ErrorF("Listener closed")
					return
				}
				blogger.Ins().ErrorF("Accept err: %v", err)
				AcceptDelay.Delay()
				continue
			}

			AcceptDelay.Reset()

			// 3.4 Handle the business method for this new connection request. At this time, the handler and conn should be bound.
			// (处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的)
			newCid := atomic.AddUint64(&s.connId, 1)
			dealConn := newServerConn(s, conn, newCid)

			go s.StartConn(dealConn)

		}
	}()
	select {
	case <-s.exitChan:
		err := listener.Close()
		if err != nil {
			blogger.Ins().ErrorF("listener close err: %v", err)
		}
	}
}

func (s *Server) ListenWebsocketConn() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Check if the server has reached the maximum allowed number of connections
		// (设置服务器最大连接控制,如果超过最大连接，则等待)
		if s.ConnMgr.Len() >= bconf.GlobalObject.MaxConnSize {
			blogger.Ins().InfoF("Exceeded the maxConnNum:%d, Wait:%d", bconf.GlobalObject.MaxConnSize, AcceptDelay.duration)
			AcceptDelay.Delay()
			return
		}
		// 2. If websocket authentication is required, set the authentication information
		// (如果需要 websocket 认证请设置认证信息)
		if s.websocketAuth != nil {
			err := s.websocketAuth(r)
			if err != nil {
				blogger.Ins().ErrorF(" websocket auth err:%v", err)
				w.WriteHeader(401)
				AcceptDelay.Delay()
				return
			}
		}
		// 3. Check if there is a subprotocol specified in the header
		// (判断 header 里面是有子协议)
		if len(r.Header.Get("Sec-Websocket-Protocol")) > 0 {
			s.upgrader.Subprotocols = websocket.Subprotocols(r)
		}
		// 4. Upgrade the connection to a websocket connection
		// (升级成 websocket 连接)
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			blogger.Ins().ErrorF("new websocket err:%v", err)
			w.WriteHeader(500)
			AcceptDelay.Delay()
			return
		}
		AcceptDelay.Reset()
		// 5. Handle the business logic of the new connection, which should already be bound to a handler and conn
		// 5. 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
		newCid := atomic.AddUint64(&s.connId, 1)
		wsConn := newWebsocketConn(s, conn, newCid)
		go s.StartConn(wsConn)

	})

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", s.IP, s.WsPort), nil)
	if err != nil {
		panic(err)
	}
}

func (s *Server) ListenKcpConn() {

	// 1. Listen to the server address
	listener, err := kcp.Listen(fmt.Sprintf("%s:%d", s.IP, s.KcpPort))
	if err != nil {
		blogger.Ins().ErrorF("[START] resolve KCP addr err: %v\n", err)
		return
	}

	blogger.Ins().InfoF("[START] KCP server listening at IP: %s, Port %d, Addr %s", s.IP, s.KcpPort, listener.Addr().String())

	// 2. Start server network connection business
	go func() {
		for {
			// 2.1 Set the maximum connection control for the server. If it exceeds the maximum connection, wait.
			// (设置服务器最大连接控制,如果超过最大连接，则等待)
			if s.ConnMgr.Len() >= bconf.GlobalObject.MaxConnSize {
				blogger.Ins().InfoF("Exceeded the maxConnNum:%d, Wait:%d", bconf.GlobalObject.MaxConnSize, AcceptDelay.duration)
				AcceptDelay.Delay()
				continue
			}
			// 2.2 Block and wait for a client to establish a connection request.
			// (阻塞等待客户端建立连接请求)
			conn, err := listener.Accept()
			if err != nil {
				blogger.Ins().ErrorF("Accept KCP err: %v", err)
				AcceptDelay.Delay()
				continue
			}

			AcceptDelay.Reset()

			// 3.4 Handle the business method for this new connection request. At this time, the handler and conn should be bound.
			// (处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn 是绑定的)
			newCid := atomic.AddUint64(&s.connId, 1)
			dealConn := newKcpServerConn(s, conn.(*kcp.UDPSession), newCid)

			go s.StartConn(dealConn)
		}
	}()
	select {
	case <-s.exitChan:
		err := listener.Close()
		if err != nil {
			blogger.Ins().ErrorF("KCP listener close err: %v", err)
		}
	}
}

// Start the network service
// (开启网络服务)
func (s *Server) Start() {
	blogger.Ins().InfoF("[START] Server name: %s,listener at IP: %s, Port %d is starting", s.Name, s.IP, s.Port)
	s.exitChan = make(chan struct{})

	// Add decoder to interceptors
	// (将解码器添加到拦截器)
	if s.decoder != nil {
		s.msgHandler.AddInterceptor(s.decoder)
	}
	// Start worker pool mechanism
	// (启动worker工作池机制)
	s.msgHandler.StartWorkerPool()

	// Start a goroutine to handle server listener business
	// (开启一个go去做服务端Listener业务)
	switch bconf.GlobalObject.Mode {
	case bconf.ServerModeTcp:
		go s.ListenTcpConn()
	case bconf.ServerModeWebsocket:
		go s.ListenWebsocketConn()
	case bconf.ServerModeKcp:
		go s.ListenKcpConn()
	default:
		go s.ListenTcpConn()
		go s.ListenWebsocketConn()
	}

}

// Stop stops the server (停止服务)
func (s *Server) Stop() {
	blogger.Ins().InfoF("[STOP] btgo server , name %s", s.Name)

	// Clear other connection information or other information that needs to be cleaned up
	// (将其他需要清理的连接信息或者其他信息 也要一并停止或者清理)
	s.ConnMgr.ClearAll()
	s.exitChan <- struct{}{}
	close(s.exitChan)
}

// Serve runs the server (运行服务)
func (s *Server) Serve() {
	s.Start()
	// Block, otherwise the listener's goroutine will exit when the main Go exits (阻塞,否则主Go退出， listenner的go将会退出)
	c := make(chan os.Signal, 1)
	// Listen for specified signals: ctrl+c or kill signal (监听指定信号 ctrl+c kill信号)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	sig := <-c
	blogger.Ins().InfoF("[SERVE] btgo server , name %s, Serve Interrupt, signal = %v", s.Name, sig)
}

func (s *Server) AddRouter(msgID uint32, router biface.IRouter) {
	if s.RouterSlicesMode {
		panic("Server RouterSlicesMode is true ")
	}
	s.msgHandler.AddRouter(msgID, router)
}

func (s *Server) AddRouterSlices(msgID uint32, router ...biface.RouterHandler) biface.IRouterSlices {
	if !s.RouterSlicesMode {
		panic("Server RouterSlicesMode is false ")
	}
	return s.msgHandler.AddRouterSlices(msgID, router...)
}

func (s *Server) Group(start, end uint32, Handlers ...biface.RouterHandler) biface.IGroupRouterSlices {
	if !s.RouterSlicesMode {
		panic("Server RouterSlicesMode is false")
	}
	return s.msgHandler.Group(start, end, Handlers...)
}

func (s *Server) Use(Handlers ...biface.RouterHandler) biface.IRouterSlices {
	if !s.RouterSlicesMode {
		panic("Server RouterSlicesMode is false")
	}
	return s.msgHandler.Use(Handlers...)
}

func (s *Server) GetConnMgr() biface.IConnManager {
	return s.ConnMgr
}

func (s *Server) SetOnConnStart(hookFunc func(biface.IConnection)) {
	s.onConnStart = hookFunc
}

func (s *Server) SetOnConnStop(hookFunc func(biface.IConnection)) {
	s.onConnStop = hookFunc
}

func (s *Server) GetOnConnStart() func(biface.IConnection) {
	return s.onConnStart
}

func (s *Server) GetOnConnStop() func(biface.IConnection) {
	return s.onConnStop
}

func (s *Server) GetPacket() biface.IDataPack {
	return s.packet
}

func (s *Server) SetPacket(packet biface.IDataPack) {
	s.packet = packet
}

func (s *Server) GetMsgHandler() biface.IMsgHandler {
	return s.msgHandler
}

// StartHeartBeat starts the heartbeat check.
// interval is the time interval between each heartbeat.
// (启动心跳检测
// interval 每次发送心跳的时间间隔)
func (s *Server) StartHeartBeat(interval time.Duration) {
	checker := NewHeartbeatChecker(interval)

	// Add the heartbeat check router. (添加心跳检测的路由)
	//检测当前路由模式
	if s.RouterSlicesMode {
		s.AddRouterSlices(checker.MsgId(), checker.RouterSlices()...)
	} else {
		s.AddRouter(checker.MsgId(), checker.Router())
	}

	// Bind the heartbeat checker to the server. (server绑定心跳检测器)
	s.hc = checker
}

// StartHeartBeatWithOption starts the heartbeat detection with the given configuration.
// interval is the time interval for sending heartbeat messages.
// option is the configuration for heartbeat detection.
// 启动心跳检测
// (option 心跳检测的配置)
func (s *Server) StartHeartBeatWithOption(interval time.Duration, option *biface.HeartBeatOption) {
	checker := NewHeartbeatChecker(interval)

	// Configure the heartbeat checker with the provided options
	if option != nil {
		checker.SetHeartbeatMsgFunc(option.MakeMsg)
		checker.SetOnRemoteNotAlive(option.OnRemoteNotAlive)
		//检测当前路由模式
		if s.RouterSlicesMode {
			checker.BindRouterSlices(option.HeartBeatMsgID, option.RouterSlices...)
		} else {
			checker.BindRouter(option.HeartBeatMsgID, option.Router)
		}
	}

	// Add the heartbeat checker's router to the server's router (添加心跳检测的路由)
	//检测当前路由模式
	if s.RouterSlicesMode {
		s.AddRouterSlices(checker.MsgId(), checker.RouterSlices()...)
	} else {
		s.AddRouter(checker.MsgId(), checker.Router())
	}

	// Bind the server with the heartbeat checker (server绑定心跳检测器)
	s.hc = checker
}

func (s *Server) GetHeartBeat() biface.IHeartbeatChecker {
	return s.hc
}

func (s *Server) SetDecoder(decoder biface.IDecoder) {
	s.decoder = decoder
}

func (s *Server) GetLengthField() *biface.LengthField {
	if s.decoder != nil {
		return s.decoder.GetLengthField()
	}
	return nil
}

func (s *Server) AddInterceptor(interceptor biface.IInterceptor) {
	s.msgHandler.AddInterceptor(interceptor)
}

func (s *Server) SetWebsocketAuth(f func(r *http.Request) error) {
	s.websocketAuth = f
}

func (s *Server) ServerName() string {
	return s.Name
}

func init() {}
