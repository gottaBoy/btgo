package bnet

import (
	"btgo/bconf"
	"btgo/biface"
	"btgo/blogger"
	"encoding/hex"
	"fmt"
	"sync"
)

const (
	// If the Worker goroutine pool is not started, a virtual workerId is assigned to the MsgHandler, which is 0, for metric counting
	// After starting the Worker goroutine pool, the ID of each worker is 0,1,2,3...
	// (如果不启动Worker协程池，则会给MsgHandler分配一个虚拟的workerId，这个workerId为0, 便于指标统计
	// 启动了Worker协程池后，每个worker的ID为0,1,2,3...)
	workerIdWithoutWorkerPool int = 0
)

// MsgHandler is the module for handling message processing callbacks
// (对消息的处理回调模块)
type MsgHandler struct {
	// A map property that stores the processing methods for each msgId
	// (存放每个msgId 所对应的处理方法的map属性)
	Routers map[uint32]biface.IRouter

	// The number of worker goroutines in the business work Worker pool
	// (业务工作Worker池的数量)
	WorkerPoolSize uint32

	// A collection of idle workers, used for bconf.WorkerModeBind
	// 空闲worker集合，用于bconf.WorkerModeBind
	freeWorkers  map[uint32]struct{}
	freeWorkerMu sync.Mutex

	// A message queue for workers to take tasks
	// (Worker负责取任务的消息队列)
	TaskQueue []chan biface.IRequest

	// Chain builder for the responsibility chain
	// (责任链构造器)
	builder      *chainBuilder
	RouterSlices *RouterSlices
}

func NewMsgHandler() *MsgHandler {
	var freeWorkers map[uint32]struct{}
	if bconf.GlobalObject.WorkerMode == bconf.WorkerModeBind {
		// Assign a workder to each link, avoid interactions when multiple links are processed by the same worker
		// MaxWorkerTaskLen can also be reduced, for example, 50
		// 为每个链接分配一个workder，避免同一worker处理多个链接时的互相影响
		// 同时可以减小MaxWorkerTaskLen，比如50，因为每个worker的负担减轻了
		bconf.GlobalObject.WorkerPoolSize = uint32(bconf.GlobalObject.MaxConnSize)
		freeWorkers = make(map[uint32]struct{}, bconf.GlobalObject.WorkerPoolSize)
		for i := uint32(0); i < bconf.GlobalObject.WorkerPoolSize; i++ {
			freeWorkers[i] = struct{}{}
		}
	}

	handler := &MsgHandler{
		Routers:        make(map[uint32]biface.IRouter),
		RouterSlices:   NewRouterSlices(),
		WorkerPoolSize: bconf.GlobalObject.WorkerPoolSize,
		// One worker corresponds to one queue (一个worker对应一个queue)
		TaskQueue:   make([]chan biface.IRequest, bconf.GlobalObject.WorkerPoolSize),
		freeWorkers: freeWorkers,
		builder:     newChainBuilder(),
	}

	// It is necessary to add the MsgHandler to the responsibility chain here, and it is the last link in the responsibility chain. After decoding in the MsgHandler, data distribution is done by router
	// (此处必须把 MsgHandler 添加到责任链中，并且是责任链最后一环，在MsgHandler中进行解码后由router做数据分发)
	handler.builder.Tail(handler)
	return handler
}

// Use worker ID
// 占用workerId
func useWorker(conn biface.IConnection) uint32 {
	var workerId uint32

	mh, _ := conn.GetMsgHandler().(*MsgHandler)
	if mh == nil {
		blogger.Ins().ErrorF("useWorker failed, mh is nil")
		return 0
	}

	if bconf.GlobalObject.WorkerMode == bconf.WorkerModeBind {
		mh.freeWorkerMu.Lock()
		defer mh.freeWorkerMu.Unlock()

		for k := range mh.freeWorkers {
			delete(mh.freeWorkers, k)
			return k
		}
	}

	//Compatible with the situation where the client has no worker, and solve the situation divide 0
	//(兼容client没有worker情况，解决除0的情况)
	if mh.WorkerPoolSize == 0 {
		workerId = 0
	} else {
		// Assign the worker responsible for processing the current connection based on the ConnID
		// Using a round-robin average allocation rule to get the workerId that needs to process this connection
		// (根据ConnID来分配当前的连接应该由哪个worker负责处理
		// 轮询的平均分配法则
		// 得到需要处理此条连接的workerId)
		workerId = uint32(conn.GetConnId() % uint64(mh.WorkerPoolSize))
	}

	return workerId
}

// Free worker ID
// 释放workerId
func freeWorker(conn biface.IConnection) {
	mh, _ := conn.GetMsgHandler().(*MsgHandler)
	if mh == nil {
		blogger.Ins().ErrorF("useWorker failed, mh is nil")
		return
	}

	if bconf.GlobalObject.WorkerMode == bconf.WorkerModeBind {
		mh.freeWorkerMu.Lock()
		defer mh.freeWorkerMu.Unlock()

		mh.freeWorkers[conn.GetWorkerId()] = struct{}{}
	}
}

// Data processing interceptor that is necessary by default in btgo
// (btgo默认必经的数据处理拦截器)
func (mh *MsgHandler) Intercept(chain biface.IChain) biface.IcResp {
	request := chain.Request()
	if request != nil {
		switch request.(type) {
		case biface.IRequest:
			iRequest := request.(biface.IRequest)
			if bconf.GlobalObject.WorkerPoolSize > 0 {
				// If the worker pool mechanism has been started, hand over the message to the worker for processing
				// (已经启动工作池机制，将消息交给Worker处理)
				mh.SendMsgToTaskQueue(iRequest)
			} else {

				// Execute the corresponding Handle method from the bound message and its corresponding processing method
				// (从绑定好的消息和对应的处理方法中执行对应的Handle方法)
				if !bconf.GlobalObject.RouterSlicesMode {
					go mh.doMsgHandler(iRequest, workerIdWithoutWorkerPool)
				} else if bconf.GlobalObject.RouterSlicesMode {
					go mh.doMsgHandlerSlices(iRequest, workerIdWithoutWorkerPool)
				}

			}
		}
	}

	return chain.Proceed(chain.Request())
}

func (mh *MsgHandler) AddInterceptor(interceptor biface.IInterceptor) {
	if mh.builder != nil {
		mh.builder.AddInterceptor(interceptor)
	}
}

// SendMsgToTaskQueue sends the message to the TaskQueue for processing by the worker
// (将消息交给TaskQueue,由worker进行处理)
func (mh *MsgHandler) SendMsgToTaskQueue(request biface.IRequest) {
	workerId := request.GetConn().GetWorkerId()
	// blogger.Ins().DebugF("Add ConnID=%d request msgId=%d to workerId=%d", request.GetConn().GetConnId(), request.GetMsgId(), workerId)
	// Send the request message to the task queue
	mh.TaskQueue[workerId] <- request
	blogger.Ins().DebugF("SendMsgToTaskQueue-->%s", hex.EncodeToString(request.GetData()))
}

// doFuncHandler handles functional requests (执行函数式请求)
func (mh *MsgHandler) doFuncHandler(request biface.IFuncRequest, workerId int) {
	defer func() {
		if err := recover(); err != nil {
			blogger.Ins().ErrorF("workerId: %d doFuncRequest panic: %v", workerId, err)
		}
	}()
	// Execute the functional request (执行函数式请求)
	request.CallFunc()
}

// doMsgHandler immediately handles messages in a non-blocking manner
// (立即以非阻塞方式处理消息)
func (mh *MsgHandler) doMsgHandler(request biface.IRequest, workerId int) {
	defer func() {
		if err := recover(); err != nil {
			blogger.Ins().ErrorF("workerId: %d doMsgHandler panic: %v", workerId, err)
		}
	}()

	msgId := request.GetMsgId()
	handler, ok := mh.Routers[msgId]

	if !ok {
		blogger.Ins().ErrorF("api msgId = %d is not FOUND!", request.GetMsgId())
		return
	}

	// Bind the Request request to the corresponding Router relationship
	// (Request请求绑定Router对应关系)
	request.BindRouter(handler)

	// Execute the corresponding processing method
	request.Call()
}

func (mh *MsgHandler) Execute(request biface.IRequest) {
	// Pass the message to the responsibility chain to handle it through interceptors layer by layer and pass it on layer by layer.
	// (将消息丢到责任链，通过责任链里拦截器层层处理层层传递)
	mh.builder.Execute(request)
}

// AddRouter adds specific processing logic for messages
// (为消息添加具体的处理逻辑)
func (mh *MsgHandler) AddRouter(msgId uint32, router biface.IRouter) {
	// 1. Check whether the current API processing method bound to the msgId already exists
	// (判断当前msg绑定的API处理方法是否已经存在)
	if _, ok := mh.Routers[msgId]; ok {
		msgErr := fmt.Sprintf("repeated api , msgId = %+v\n", msgId)
		panic(msgErr)
	}
	// 2. Add the binding relationship between msg and API
	// (添加msg与api的绑定关系)
	mh.Routers[msgId] = router
	blogger.Ins().InfoF("Add Router msgId = %d", msgId)
}

// AddRouterSlices adds router handlers using slices
// (切片路由添加)
func (mh *MsgHandler) AddRouterSlices(msgId uint32, handler ...biface.RouterHandler) biface.IRouterSlices {
	mh.RouterSlices.AddHandler(msgId, handler...)
	return mh.RouterSlices
}

// Group routes into a group (路由分组)
func (mh *MsgHandler) Group(start, end uint32, Handlers ...biface.RouterHandler) biface.IGroupRouterSlices {
	return NewGroup(start, end, mh.RouterSlices, Handlers...)
}
func (mh *MsgHandler) Use(Handlers ...biface.RouterHandler) biface.IRouterSlices {
	mh.RouterSlices.Use(Handlers...)
	return mh.RouterSlices
}

func (mh *MsgHandler) doMsgHandlerSlices(request biface.IRequest, workerId int) {
	defer func() {
		if err := recover(); err != nil {
			blogger.Ins().ErrorF("workerId: %d doMsgHandler panic: %v", workerId, err)
		}
	}()

	msgId := request.GetMsgId()
	handlers, ok := mh.RouterSlices.GetHandlers(msgId)
	if !ok {
		blogger.Ins().ErrorF("api msgId = %d is not FOUND!", request.GetMsgId())
		return
	}

	request.BindRouterSlices(handlers)
	request.RouterSlicesNext()
}

// StartOneWorker starts a worker workflow
// (启动一个Worker工作流程)
func (mh *MsgHandler) StartOneWorker(workerId int, taskQueue chan biface.IRequest) {
	blogger.Ins().InfoF("Worker ID = %d is started.", workerId)
	// Continuously wait for messages in the queue
	// (不断地等待队列中的消息)
	for {
		select {
		// If there is a message, take out the Request from the queue and execute the bound business method
		// (有消息则取出队列的Request，并执行绑定的业务方法)
		case request := <-taskQueue:

			switch req := request.(type) {

			case biface.IFuncRequest:
				// Internal function call request (内部函数调用request)

				mh.doFuncHandler(req, workerId)

			case biface.IRequest: // Client message request

				if !bconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandler(req, workerId)
				} else if bconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandlerSlices(req, workerId)
				}
			}
		}
	}
}

// StartWorkerPool starts the worker pool
func (mh *MsgHandler) StartWorkerPool() {
	// Iterate through the required number of workers and start them one by one
	// (遍历需要启动worker的数量，依此启动)
	for i := 0; i < int(mh.WorkerPoolSize); i++ {
		// A worker is started
		// Allocate space for the corresponding task queue for the current worker
		// (给当前worker对应的任务队列开辟空间)
		mh.TaskQueue[i] = make(chan biface.IRequest, bconf.GlobalObject.MaxWorkerTaskLen)

		// Start the current worker, blocking and waiting for messages to be passed in the corresponding task queue
		// (启动当前Worker，阻塞的等待对应的任务队列是否有消息传递进来)
		go mh.StartOneWorker(i, mh.TaskQueue[i])
	}
}
