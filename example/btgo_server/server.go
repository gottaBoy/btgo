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

// 创建连接的时候执行
func DoConnStart(conn biface.IConnection) {
	fmt.Println("DoConnStart is Called ... ")
	fmt.Println("Set conn Name, Home done!")
	conn.SetProperty("Name", "gottabay")
	conn.SetProperty("Home", "https://github.com/gottaboy/btgo")
	err := conn.SendMsg(0, []byte("DoConnStart ..."))
	if err != nil {
		fmt.Println(err)
	}
}

// 连接断开的时候执行
func DoConnClose(conn biface.IConnection) {
	if name, err := conn.GetProperty("Name"); err == nil {
		fmt.Println("Conn Property Name = ", name)
	}

	if home, err := conn.GetProperty("Home"); err == nil {
		fmt.Println("Conn Property Home = ", home)
	}
	fmt.Println("DoConnClose is Called ... ")
}

func main() {
	s := bnet.NewServer()
	s.SetOnConnStart(DoConnStart)
	s.SetOnConnStop(DoConnClose)
	s.AddRouter(0, &PingRouter{})
	s.Serve()
}
