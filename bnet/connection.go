package bnet

import (
	"btgo/biface"
	"fmt"
	"net"
)

type Connection struct {
	Conn     *net.TCPConn
	ConnId   uint32
	isClosed bool
	// handler     biface.HandlerFunc
	Router      biface.IRouter
	ExitBufChan chan bool
}

func NewConn(conn *net.TCPConn, connId uint32, router biface.IRouter) *Connection {
	return &Connection{
		Conn:     conn,
		ConnId:   connId,
		isClosed: false,
		// handler:     handler,
		Router:      router,
		ExitBufChan: make(chan bool, 1),
	}
}

func (c *Connection) Open() {
	go c.StartReader()
	for {
		select {
		case <-c.ExitBufChan:
			return
		}
	}
}

func (c *Connection) Close() {
	if c.isClosed == false {
		return
	}
	// 设置关闭
	c.isClosed = true

	//TODO callback

	// 关闭
	c.Conn.Close()

	// 通知退出
	c.ExitBufChan <- true

	// 关闭通道
	close(c.ExitBufChan)
}

func (c *Connection) IsClose() bool {
	return (c.isClosed == true)
}

func (c *Connection) StartReader() {
	fmt.Println(c.ConnId, " conn start read!")
	defer fmt.Println(c.GetAdrr(), "conn is exited!")
	defer c.Close()
	for {
		// 读取请求数据
		buf := make([]byte, 512)
		_, err := c.Conn.Read(buf)
		if err != nil {
			fmt.Println("Recv buf err ", err)
			c.ExitBufChan <- true
			continue
		}
		// 处理请求
		req := Request{
			conn: c,
			data: buf,
		}
		go func(request biface.IRequest) {
			c.Router.PreHandler(request)
			c.Router.Handler(request)
			c.Router.PostHandler(request)
		}(&req)
		// if err := c.handler(c.Conn, buf, cnt); err != nil {
		// 	fmt.Println("ConnId ", c.ConnId, "handler is error!")
		// 	c.ExitBufChan <- true
		// 	return
		// }
	}
}

func (c *Connection) GetAdrr() net.Addr {
	return c.Conn.RemoteAddr()
}

func (c *Connection) GetConn() *net.TCPConn {
	return c.Conn
}

func (c *Connection) GetConnId() uint32 {
	return c.ConnId
}
