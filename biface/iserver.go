package biface

type IServer interface {
	Start()
	Serve()
	Stop()
	AddRouter(msgId uint32, router IRouter)
}
