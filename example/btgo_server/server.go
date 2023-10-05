package main

import (
	"btgo/biface"
	"btgo/bnet"
	"fmt"
)

type PingRouter struct {
	bnet.BaseRouter
}

func (pr *PingRouter) PreHandler(request biface.IRequest) {
	fmt.Println("Call Router PreHandler")
	// _, err := request.GetConn().GetConn().Write([]byte("before ping ... \n"))
	err := request.GetConn().SendMsg(request.GetMsgId(), []byte("before ping ... \n"))
	if err != nil {
		fmt.Println("call back ping ping ping error")
	}
}

func (pr *PingRouter) Handler(request biface.IRequest) {
	fmt.Println("Call PingRouter Handler")
	fmt.Println("recv from client : msgId=", request.GetMsgId(), ", data=", string(request.GetData()))
	// _, err := request.GetConn().GetConn().Write([]byte("ping...ping...ping \n"))
	err := request.GetConn().SendMsg(request.GetMsgId(), []byte("ping...ping...ping \n"))
	if err != nil {
		fmt.Println("call back ping ping ping error")
	}
}

func (pr *PingRouter) PostHandler(request biface.IRequest) {
	fmt.Println("Call Router PostHandler")
	// _, err := request.GetConn().GetConn().Write([]byte("After ping ..... \n"))
	err := request.GetConn().SendMsg(request.GetMsgId(), []byte("After ping ..... \n"))
	if err != nil {
		fmt.Println("call back ping ping ping error")
	}
}

func main() {
	server := bnet.NewServer("[BTGO Server]")
	server.AddRouter(0, &PingRouter{})
	server.Serve()
}
