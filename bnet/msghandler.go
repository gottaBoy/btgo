package bnet

import (
	"btgo/biface"
	"btgo/utils"
	"fmt"
	"strconv"
)

type MsgHandler struct {
	Routers        map[uint32]biface.IRouter
	WorkerPoolSize uint32
	TaskQueue      []chan biface.IRequest
}

func NewMsgHandler() *MsgHandler {
	return &MsgHandler{
		Routers:        make(map[uint32]biface.IRouter),
		WorkerPoolSize: utils.GlobalObject.WorkerPoolSize,
		TaskQueue:      make([]chan biface.IRequest, utils.GlobalObject.WorkerPoolSize),
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

func (mh *MsgHandler) StartOneWorker(workerId int, taskQueue chan biface.IRequest) {
	fmt.Println("Worker id = ", workerId, " is started.")
	//不断的等待队列中的消息
	for {
		select {
		//有消息则取出队列的Request，并执行绑定的业务方法
		case request := <-taskQueue:
			mh.DoMsgHandler(request)
		}
	}
}

func (mh *MsgHandler) StartWorkerPool() {
	for i := 0; i < int(mh.WorkerPoolSize); i++ {
		mh.TaskQueue[i] = make(chan biface.IRequest, utils.GlobalObject.MaxWorkerTask)
		go mh.StartOneWorker(i, mh.TaskQueue[i])
	}
}

func (mh *MsgHandler) SendMsgToTaskQueue(request biface.IRequest) {
	workerId := request.GetConn().GetConnId() % mh.WorkerPoolSize
	fmt.Println("Add ConnId=", request.GetConn().GetConnId(), " request msgId=", request.GetMsgId(), "to workerId=", workerId)
	//将请求消息发送给任务队列
	mh.TaskQueue[workerId] <- request
}
