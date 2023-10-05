package bnet

import (
	"btgo/biface"
	"btgo/utils"
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
	// Router      biface.IRouter
	MsgHandler  biface.IMsgHandler
	ExitBufChan chan bool
	msgChan     chan []byte
}

func NewConn(conn *net.TCPConn, connId uint32, msgHandler biface.IMsgHandler) *Connection {
	return &Connection{
		Conn:     conn,
		ConnId:   connId,
		isClosed: false,
		// handler:     handler,
		// Router:      router,
		MsgHandler:  msgHandler,
		ExitBufChan: make(chan bool, 1),
		msgChan:     make(chan []byte),
	}
}

func (c *Connection) Open() {
	go c.StartReader()
	go c.StartWriter()
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
		// go func(request biface.IRequest) {
		// 	c.Router.PreHandler(request)
		// 	c.Router.Handler(request)
		// 	c.Router.PostHandler(request)
		// }(&req)
		if utils.GlobalObject.WorkerPoolSize > 0 {
			//已经启动工作池机制，将消息交给Worker处理
			c.MsgHandler.SendMsgToTaskQueue(&req)
		} else {
			//从绑定好的消息和对应的处理方法中执行对应的Handle方法
			go c.MsgHandler.DoMsgHandler(&req)
		}
	}
}

func (c *Connection) StartWriter() {
	fmt.Println(c.ConnId, " conn start writer!")
	defer fmt.Println(c.GetAdrr(), "conn writer is exited!")
	for {
		select {
		case data := <-c.msgChan:
			if _, err := c.Conn.Write(data); err != nil {
				fmt.Println("Send Data error:, ", err, " Conn Writer exit")
				return
			}
		case <-c.ExitBufChan:
			return
		}
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
	// if _, err := c.Conn.Write(msg); err != nil {
	// 	fmt.Println("Write msg id ", msgId, " error ")
	// 	c.ExitBufChan <- true
	// 	return errors.New("conn Write error")
	// }
	c.msgChan <- msg

	return nil
}
