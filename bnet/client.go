package bnet

import (
	"btgo/bconf"
	"btgo/bdecoder"
	"btgo/biface"
	"btgo/blogger"
	"btgo/bpack"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	// Client Name 客户端的名称
	Name string
	// IP of the target server to connect 目标链接服务器的IP
	Ip string
	// Port of the target server to connect 目标链接服务器的端口
	Port int
	// Client version tcp,websocket,客户端版本 tcp,websocket
	version string
	// Connection instance 链接实例
	conn biface.IConnection
	// Hook function called on connection start 该client的连接创建时Hook函数
	onConnStart func(conn biface.IConnection)
	// Hook function called on connection stop 该client的连接断开时的Hook函数
	onConnStop func(conn biface.IConnection)
	// Data packet packer 数据报文封包方式
	packet biface.IDataPack
	// Asynchronous channel for capturing connection close status 异步捕获链接关闭状态
	exitChan chan struct{}
	// Message management module 消息管理模块
	msgHandler biface.IMsgHandler
	// Disassembly and assembly decoder for resolving sticky and broken packages
	//断粘包解码器
	decoder biface.IDecoder
	// Heartbeat checker 心跳检测器
	hc biface.IHeartbeatChecker
	// Use TLS 使用TLS
	useTLS bool
	// For websocket connections
	dialer *websocket.Dialer
	// Error channel
	ErrChan chan error
}

func NewClient(ip string, port int, opts ...ClientOption) biface.IClient {

	c := &Client{
		// Default name, can be modified using the WithNameClient Option
		// (默认名称，可以使用WithNameClient的Option修改)
		Name: "BtgoClientTcp",
		Ip:   ip,
		Port: port,

		msgHandler: NewMsgHandler(),
		packet:     bpack.Factory().NewPack(biface.BtgoDataPack), // Default to using Btgo's TLV packet format(默认使用Btgo的TLV封包方式)
		decoder:    bdecoder.NewTLVDecoder(),                     // Default to using Btgo's TLV decoder(默认使用Btgo的TLV解码器)
		version:    "tcp",
		ErrChan:    make(chan error),
	}

	// Apply Option settings (应用Option设置)
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func NewWsClient(ip string, port int, opts ...ClientOption) biface.IClient {

	c := &Client{
		// Default name, can be modified using the WithNameClient Option
		// (默认名称，可以使用WithNameClient的Option修改)
		Name: "BtgoClientWs",
		Ip:   ip,
		Port: port,

		msgHandler: NewMsgHandler(),
		packet:     bpack.Factory().NewPack(biface.BtgoDataPack), // Default to using Btgo's TLV packet format(默认使用Btgo的TLV封包方式)
		decoder:    bdecoder.NewTLVDecoder(),                     // Default to using Btgo's TLV decoder(默认使用Btgo的TLV解码器)
		version:    "websocket",
		dialer:     &websocket.Dialer{},
		ErrChan:    make(chan error),
	}

	// Apply Option settings (应用Option设置)
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func NewTLSClient(ip string, port int, opts ...ClientOption) biface.IClient {

	c, _ := NewClient(ip, port, opts...).(*Client)

	c.useTLS = true

	return c
}

// Start starts the client, sends requests and establishes a connection.
// (重新启动客户端，发送请求且建立连接)
func (c *Client) Restart() {
	c.exitChan = make(chan struct{})

	// Set worker pool size to 0 to turn off the worker pool in the client (客户端将协程池关闭)
	bconf.GlobalObject.WorkerPoolSize = 0

	go func() {

		addr := &net.TCPAddr{
			IP:   net.ParseIP(c.Ip),
			Port: c.Port,
			Zone: "", //for ipv6, ignore
		}

		// Create a raw socket and get net.Conn (创建原始Socket，得到net.Conn)
		switch c.version {
		case "websocket":
			wsAddr := fmt.Sprintf("ws://%s:%d", c.Ip, c.Port)

			// Create a raw socket and get net.Conn (创建原始Socket，得到net.Conn)
			wsConn, _, err := c.dialer.Dial(wsAddr, nil)
			if err != nil {
				// connection failed
				blogger.Ins().ErrorF("WsClient connect to server failed, err:%v", err)
				c.ErrChan <- err
				return
			}
			// Create Connection object
			c.conn = newWsClientConn(c, wsConn)

		default:
			var conn net.Conn
			var err error
			if c.useTLS {
				// TLS encryption
				config := &tls.Config{
					// Skip certificate verification here because the CA certificate of the certificate issuer is not authenticated
					// (这里是跳过证书验证，因为证书签发机构的CA证书是不被认证的)
					InsecureSkipVerify: true,
				}

				conn, err = tls.Dial("tcp", fmt.Sprintf("%v:%v", net.ParseIP(c.Ip), c.Port), config)
				if err != nil {
					blogger.Ins().ErrorF("tls client connect to server failed, err:%v", err)
					c.ErrChan <- err
					return
				}
			} else {
				conn, err = net.DialTCP("tcp", nil, addr)
				if err != nil {
					// connection failed
					blogger.Ins().ErrorF("client connect to server failed, err:%v", err)
					c.ErrChan <- err
					return
				}
			}
			// Create Connection object
			c.conn = newClientConn(c, conn)
		}

		blogger.Ins().InfoF("[START] Btgo Client LocalAddr: %s, RemoteAddr: %s\n", c.conn.LocalAddr(), c.conn.RemoteAddr())
		// HeartBeat detection
		if c.hc != nil {
			// Bind connection and heartbeat detector after connection is successfully established
			// (创建链接成功，绑定链接与心跳检测器)
			c.hc.BindConn(c.conn)
		}

		// Start connection
		go c.conn.Start()

		select {
		case <-c.exitChan:
			blogger.Ins().InfoF("client exit.")
		}
	}()
}

// Start starts the client, sends requests and establishes a connection.
// (启动客户端，发送请求且建立链接)
func (c *Client) Start() {

	// Add the decoder to the interceptor list (将解码器添加到拦截器)
	if c.decoder != nil {
		c.msgHandler.AddInterceptor(c.decoder)
	}

	c.Restart()
}

// StartHeartBeat starts heartbeat detection with a fixed time interval.
// interval: the time interval between each heartbeat message.
// (启动心跳检测, interval: 每次发送心跳的时间间隔)
func (c *Client) StartHeartBeat(interval time.Duration) {
	checker := NewHeartbeatChecker(interval)

	// Add the heartbeat checker's route to the client's message handler.
	// (添加心跳检测的路由)
	c.AddRouter(checker.MsgId(), checker.Router())

	// Bind the heartbeat checker to the client's connection.
	// (client绑定心跳检测器)
	c.hc = checker
}

// StartHeartBeatWithOption starts heartbeat detection with a custom callback function.
// interval: the time interval between each heartbeat message.
// option: a HeartBeatOption struct that contains the custom callback function and message
// 启动心跳检测(自定义回调)
func (c *Client) StartHeartBeatWithOption(interval time.Duration, option *biface.HeartBeatOption) {
	// Create a new heartbeat checker with the given interval.
	checker := NewHeartbeatChecker(interval)

	// Set the heartbeat checker's callback function and message ID based on the HeartBeatOption struct.
	if option != nil {
		checker.SetHeartbeatMsgFunc(option.MakeMsg)
		checker.SetOnRemoteNotAlive(option.OnRemoteNotAlive)
		checker.BindRouter(option.HeartBeatMsgID, option.Router)
	}

	// Add the heartbeat checker's route to the client's message handler.
	c.AddRouter(checker.MsgId(), checker.Router())

	// Bind the heartbeat checker to the client's connection.
	c.hc = checker
}

func (c *Client) Stop() {
	blogger.Ins().InfoF("[STOP] Btgo Client LocalAddr: %s, RemoteAddr: %s\n", c.conn.LocalAddr(), c.conn.RemoteAddr())
	c.conn.Stop()
	c.exitChan <- struct{}{}
	close(c.exitChan)
	close(c.ErrChan)
}

func (c *Client) AddRouter(msgId uint32, router biface.IRouter) {
	c.msgHandler.AddRouter(msgId, router)
}

func (c *Client) Conn() biface.IConnection {
	return c.conn
}

func (c *Client) SetOnConnStart(hookFunc func(biface.IConnection)) {
	c.onConnStart = hookFunc
}

func (c *Client) SetOnConnStop(hookFunc func(biface.IConnection)) {
	c.onConnStop = hookFunc
}

func (c *Client) GetOnConnStart() func(biface.IConnection) {
	return c.onConnStart
}

func (c *Client) GetOnConnStop() func(biface.IConnection) {
	return c.onConnStop
}

func (c *Client) GetPacket() biface.IDataPack {
	return c.packet
}

func (c *Client) SetPacket(packet biface.IDataPack) {
	c.packet = packet
}

func (c *Client) GetMsgHandler() biface.IMsgHandler {
	return c.msgHandler
}

func (c *Client) AddInterceptor(interceptor biface.IInterceptor) {
	c.msgHandler.AddInterceptor(interceptor)
}

func (c *Client) SetDecoder(decoder biface.IDecoder) {
	c.decoder = decoder
}
func (c *Client) GetLengthField() *biface.LengthField {
	if c.decoder != nil {
		return c.decoder.GetLengthField()
	}
	return nil
}

func (c *Client) GetErrChan() chan error {
	return c.ErrChan
}

func (c *Client) SetName(name string) {
	c.Name = name
}

func (c *Client) GetName() string {
	return c.Name
}
