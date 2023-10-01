package bnet

import (
	"btgo/biface"
	"errors"
	"fmt"
	"io"
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
		dp := NewDataPack()

		headData := make([]byte, dp.GetHeadLen())
		if _, err := io.ReadFull(c.Conn, headData); err != nil {
			fmt.Println("Read msg err ", err)
			c.ExitBufChan <- true
			continue
		}
		msg, err := dp.UnPack(headData)
		if err != nil {
			fmt.Println("unpack error ", err)
			c.ExitBufChan <- true
			continue
		}
		var data []byte
		if msg.GetDataLen() > 0 {
			data = make([]byte, msg.GetDataLen())
			if _, err := io.ReadFull(c.Conn, data); err != nil {
				fmt.Println("read msg data error ", err)
				c.ExitBufChan <- true
				continue
			}
		}
		msg.SetData(data)
		// 处理请求
		req := Request{
			conn: c,
			msg:  msg,
		}
		go func(request biface.IRequest) {
			c.Router.PreHandler(request)
			c.Router.Handler(request)
			c.Router.PostHandler(request)
		}(&req)
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

func (c *Connection) SendMsg(msgId uint32, data []byte) error {
	if c.isClosed == true {
		return errors.New("Connection closed when send msg")
	}
	// 将data封包，并且发送
	dp := NewDataPack()
	msg, err := dp.Pack(NewMessage(msgId, data))
	if err != nil {
		fmt.Println("Pack error msg id = ", msgId)
		return errors.New("Pack error msg ")
	}

	// 写回客户端
	if _, err := c.Conn.Write(msg); err != nil {
		fmt.Println("Write msg id ", msgId, " error ")
		c.ExitBufChan <- true
		return errors.New("conn Write error")
	}

	return nil
}
