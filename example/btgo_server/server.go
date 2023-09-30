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
	_, err := request.GetConn().GetConn().Write([]byte("before ping ... \n"))
	if err != nil {
		fmt.Println("call back ping ping ping error")
	}
}

func (this *PingRouter) Handler(request biface.IRequest) {
	fmt.Println("Call PingRouter Handler")
	_, err := request.GetConn().GetConn().Write([]byte("ping...ping...ping \n"))
	if err != nil {
		fmt.Println("call back ping ping ping error")
	}
}

func (this *PingRouter) PostHandler(request biface.IRequest) {
	fmt.Println("Call Router PostHandler")
	_, err := request.GetConn().GetConn().Write([]byte("After ping ..... \n"))
	if err != nil {
		fmt.Println("call back ping ping ping error")
	}
}

func main() {
	server := bnet.NewServer("[BTGO Server]")
	server.AddRouter(&PingRouter{})
	server.Serve()
}