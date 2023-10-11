package bnet

import (
	"fmt"
	"time"

	"btgo/biface"
	"btgo/blogger"
)

type HeartbeatChecker struct {
	interval time.Duration //  Heartbeat detection interval(心跳检测时间间隔)
	quitChan chan bool     // Quit signal(退出信号)

	makeMsg biface.HeartBeatMsgFunc //User-defined heartbeat message processing method(用户自定义的心跳检测消息处理方法)

	onRemoteNotAlive biface.OnRemoteNotAlive //  User-defined method for handling remote connections that are not alive (用户自定义的远程连接不存活时的处理方法)

	msgId        uint32                 // Heartbeat message ID(心跳的消息ID)
	router       biface.IRouter         // User-defined heartbeat message business processing router(用户自定义的心跳检测消息业务处理路由)
	routerSlices []biface.RouterHandler //(用户自定义的心跳检测消息业务处理新路由)
	conn         biface.IConnection     // Bound connection(绑定的链接)

	beatFunc biface.HeartBeatFunc // // User-defined heartbeat sending function(用户自定义心跳发送函数)
}

/*
Default callback routing business for receiving remote heartbeat messages
(收到remote心跳消息的默认回调路由业务)
*/
type HeatBeatDefaultRouter struct {
	BaseRouter
}

func (r *HeatBeatDefaultRouter) Handler(req biface.IRequest) {
	blogger.Ins().InfoF("Recv Heartbeat from %s, MsgId = %+v, Data = %s",
		req.GetConn().RemoteAddr(), req.GetMsgId(), string(req.GetData()))
}

func HeatBeatDefaultHandler(req biface.IRequest) {
	blogger.Ins().InfoF("Recv Heartbeat from %s, MsgId = %+v, Data = %s",
		req.GetConn().RemoteAddr(), req.GetMsgId(), string(req.GetData()))
}

func makeDefaultMsg(conn biface.IConnection) []byte {
	msg := fmt.Sprintf("heartbeat [%s->%s]", conn.LocalAddr(), conn.RemoteAddr())
	return []byte(msg)
}

func notAliveDefaultFunc(conn biface.IConnection) {
	blogger.Ins().InfoF("Remote connection %s is not alive, stop it", conn.RemoteAddr())
	conn.Stop()
}

func NewHeartbeatChecker(interval time.Duration) biface.IHeartbeatChecker {
	heartbeat := &HeartbeatChecker{
		interval: interval,
		quitChan: make(chan bool),

		// Use default heartbeat message generation function and remote connection not alive handling method
		// (均使用默认的心跳消息生成函数和远程连接不存活时的处理方法)
		makeMsg:          makeDefaultMsg,
		onRemoteNotAlive: notAliveDefaultFunc,
		msgId:            biface.HeartBeatDefaultMsgID,
		router:           &HeatBeatDefaultRouter{},
		routerSlices:     []biface.RouterHandler{HeatBeatDefaultHandler},
		beatFunc:         nil,
	}

	return heartbeat
}

func (h *HeartbeatChecker) SetOnRemoteNotAlive(f biface.OnRemoteNotAlive) {
	if f != nil {
		h.onRemoteNotAlive = f
	}
}

func (h *HeartbeatChecker) SetHeartbeatMsgFunc(f biface.HeartBeatMsgFunc) {
	if f != nil {
		h.makeMsg = f
	}
}

func (h *HeartbeatChecker) SetHeartbeatFunc(beatFunc biface.HeartBeatFunc) {
	if beatFunc != nil {
		h.beatFunc = beatFunc
	}
}

func (h *HeartbeatChecker) BindRouter(msgId uint32, router biface.IRouter) {
	if router != nil && msgId != biface.HeartBeatDefaultMsgID {
		h.msgId = msgId
		h.router = router
	}
}

func (h *HeartbeatChecker) BindRouterSlices(msgId uint32, handlers ...biface.RouterHandler) {
	if len(handlers) > 0 && msgId != biface.HeartBeatDefaultMsgID {
		h.msgId = msgId
		h.routerSlices = append(h.routerSlices, handlers...)
	}
}

func (h *HeartbeatChecker) start() {
	ticker := time.NewTicker(h.interval)
	for {
		select {
		case <-ticker.C:
			h.check()
		case <-h.quitChan:
			ticker.Stop()
			return
		}
	}
}

func (h *HeartbeatChecker) Start() {
	go h.start()
}

func (h *HeartbeatChecker) Stop() {
	blogger.Ins().InfoF("heartbeat checker stop, connID=%+v", h.conn.GetConnId())
	h.quitChan <- true
}

func (h *HeartbeatChecker) SendHeartBeatMsg() error {

	msg := h.makeMsg(h.conn)

	err := h.conn.SendMsg(h.msgId, msg)
	if err != nil {
		blogger.Ins().ErrorF("send heartbeat msg error: %v, msgId=%+v msg=%+v", err, h.msgId, msg)
		return err
	}

	return nil
}

func (h *HeartbeatChecker) check() (err error) {

	if h.conn == nil {
		return nil
	}

	if !h.conn.IsAlive() {
		h.onRemoteNotAlive(h.conn)
	} else {
		if h.beatFunc != nil {
			err = h.beatFunc(h.conn)
		} else {
			err = h.SendHeartBeatMsg()
		}
	}

	return err
}

func (h *HeartbeatChecker) BindConn(conn biface.IConnection) {
	h.conn = conn
	conn.SetHeartBeat(h)
}

// Clone clones to a specified connection
// (克隆到一个指定的链接上)
func (h *HeartbeatChecker) Clone() biface.IHeartbeatChecker {

	heartbeat := &HeartbeatChecker{
		interval:         h.interval,
		quitChan:         make(chan bool),
		beatFunc:         h.beatFunc,
		makeMsg:          h.makeMsg,
		onRemoteNotAlive: h.onRemoteNotAlive,
		msgId:            h.msgId,
		router:           h.router,
		routerSlices:     h.routerSlices,
		conn:             nil, // The bound connection needs to be reassigned
	}

	return heartbeat
}

func (h *HeartbeatChecker) MsgId() uint32 {
	return h.msgId
}

func (h *HeartbeatChecker) Router() biface.IRouter {
	return h.router
}

func (h *HeartbeatChecker) RouterSlices() []biface.RouterHandler {
	return h.routerSlices
}
