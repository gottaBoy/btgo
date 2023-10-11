package bnet

import (
	"strconv"
	"sync"

	"btgo/biface"
)

// BaseRouter is used as the base class when implementing a router.
// Depending on the needs, the methods of this base class can be overridden.
// (实现router时，先嵌入这个基类，然后根据需要对这个基类的方法进行重写)
type BaseRouter struct{}

// Here, all of BaseRouter's methods are empty, because some routers may not want to have PreHandler or PostHandler.
// Therefore, inheriting all routers from BaseRouter has the advantage that PreHandler and PostHandler do not need to be
// implemented to instantiate a router.
// (这里之所以BaseRouter的方法都为空，
// 是因为有的Router不希望有PreHandler或PostHandler
// 所以Router全部继承BaseRouter的好处是，不需要实现PreHandler和PostHandler也可以实例化)

// PreHandler -
func (br *BaseRouter) PreHandler(req biface.IRequest) {}

// Handler -
func (br *BaseRouter) Handler(req biface.IRequest) {}

// PostHandler -
func (br *BaseRouter) PostHandler(req biface.IRequest) {}

// New slice-based router
// The new version of the router has basic logic that allows users to pass in varying numbers of router Handlers.
// The router will save all of these router Handlerr functions and find them when a request comes in, then execute them using IRequest.
// The router can set globally shared components using the Use method.
// The router can be grouped using Group, and groups also have their own Use method for setting group-shared components.
// (新切片集合式路由
// 新版本路由基本逻辑,用户可以传入不等数量的路由路由处理器
// 路由本体会讲这些路由处理器函数全部保存,在请求来的时候找到，并交由IRequest去执行
// 路由可以设置全局的共用组件通过Use方法
// 路由可以分组,通过Group,分组也有自己对应Use方法设置组共有组件)

type RouterSlices struct {
	Routers  map[uint32][]biface.RouterHandler
	Handlers []biface.RouterHandler
	sync.RWMutex
}

func NewRouterSlices() *RouterSlices {
	return &RouterSlices{
		Routers:  make(map[uint32][]biface.RouterHandler, 10),
		Handlers: make([]biface.RouterHandler, 0, 6),
	}
}

func (r *RouterSlices) Use(Handlers ...biface.RouterHandler) {
	r.Handlers = append(r.Handlers, Handlers...)
}

func (r *RouterSlices) AddHandler(msgId uint32, Handlers ...biface.RouterHandler) {
	// 1. Check if the API Handlerr method bound to the current msg already exists
	if _, ok := r.Routers[msgId]; ok {
		panic("repeated api , msgId = " + strconv.Itoa(int(msgId)))
	}

	finalSize := len(r.Handlers) + len(Handlers)
	mergedHandlers := make([]biface.RouterHandler, finalSize)
	copy(mergedHandlers, r.Handlers)
	copy(mergedHandlers[len(r.Handlers):], Handlers)
	r.Routers[msgId] = append(r.Routers[msgId], mergedHandlers...)
}

func (r *RouterSlices) GetHandlers(MsgId uint32) ([]biface.RouterHandler, bool) {
	r.RLock()
	defer r.RUnlock()
	Handlers, ok := r.Routers[MsgId]
	return Handlers, ok
}

func (r *RouterSlices) Group(start, end uint32, Handlers ...biface.RouterHandler) biface.IGroupRouterSlices {
	return NewGroup(start, end, r, Handlers...)
}

type GroupRouter struct {
	start    uint32
	end      uint32
	Handlers []biface.RouterHandler
	router   biface.IRouterSlices
}

func NewGroup(start, end uint32, router *RouterSlices, Handlers ...biface.RouterHandler) *GroupRouter {
	g := &GroupRouter{
		start:    start,
		end:      end,
		Handlers: make([]biface.RouterHandler, 0, len(Handlers)),
		router:   router,
	}
	g.Handlers = append(g.Handlers, Handlers...)
	return g
}

func (g *GroupRouter) Use(Handlers ...biface.RouterHandler) {
	g.Handlers = append(g.Handlers, Handlers...)
}

func (g *GroupRouter) AddHandler(MsgId uint32, Handlers ...biface.RouterHandler) {
	if MsgId < g.start || MsgId > g.end {
		panic("add router to group err in msgId:" + strconv.Itoa(int(MsgId)))
	}

	finalSize := len(g.Handlers) + len(Handlers)
	mergedHandlers := make([]biface.RouterHandler, finalSize)
	copy(mergedHandlers, g.Handlers)
	copy(mergedHandlers[len(g.Handlers):], Handlers)

	g.router.AddHandler(MsgId, mergedHandlers...)
}
