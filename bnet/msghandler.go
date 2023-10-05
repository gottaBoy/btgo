package bnet

import (
	"btgo/biface"
	"fmt"
	"strconv"
)

type MsgHandler struct {
	Routers map[uint32]biface.IRouter
}

func NewMsgHandler() *MsgHandler {
	return &MsgHandler{
		Routers: make(map[uint32]biface.IRouter),
	}
}

func (mh *MsgHandler) DoMsgHandler(request biface.IRequest) {
	handler, ok := mh.Routers[request.GetMsgId()]
	if !ok {
		fmt.Println("api msgId = ", request.GetMsgId(), " is not FOUND!")
		return
	}

	// 执行对应处理方法
	handler.PreHandler(request)
	handler.Handler(request)
	handler.PostHandler(request)
}

func (mh *MsgHandler) AddRouter(msgId uint32, router biface.IRouter) {
	if _, ok := mh.Routers[msgId]; ok {
		panic("repeated router , msgId = " + strconv.Itoa(int(msgId)))
	}
	mh.Routers[msgId] = router
	fmt.Println("Add router msgId = ", msgId)
}
