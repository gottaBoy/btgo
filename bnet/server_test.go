package bnet

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"btgo/biface"
	"btgo/bpack"
)

// run in terminal:
// go test -v ./bnet -run=TestServer

/*
ClientTest client
*/
func ClientTest(i uint32) {

	fmt.Println("Client Test ... start")

	//3秒之后发起测试请求，给服务端开启服务的机会
	time.Sleep(3 * time.Second)

	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Println("client start err, exit!")
		return
	}

	for {
		dp := bpack.Factory().NewPack(biface.BtgoDataPack)
		msg, _ := dp.Pack(bpack.NewMsgPackage(i, []byte("client test message")))
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("client write err: ", err)
			return
		}

		headData := make([]byte, dp.GetHeadLen())
		_, err = io.ReadFull(conn, headData)
		if err != nil {
			fmt.Println("client read head err: ", err)
			return
		}

		msgHead, err := dp.UnPack(headData)
		if err != nil {
			fmt.Println("client unpack head err: ", err)
			return
		}

		if msgHead.GetDataLen() > 0 {
			msg := msgHead.(*bpack.Message)
			msg.Data = make([]byte, msg.GetDataLen())

			_, err := io.ReadFull(conn, msg.Data)
			if err != nil {
				fmt.Println("client unpack data err")
				return
			}

			fmt.Printf("==> Client receive Msg: ID = %d, len = %d , data = %s\n", msg.Id, msg.DataLen, msg.Data)
		}

		time.Sleep(time.Second)
	}
}

/*
	server
*/

type PingRouter struct {
	BaseRouter
}

// Test PreHandler
func (pr *PingRouter) PreHandler(request biface.IRequest) {
	fmt.Println("Call Router PreHandler")
	err := request.GetConn().SendMsg(1, []byte("before ping ....\n"))
	if err != nil {
		fmt.Println("preHandle SendMsg err: ", err)
	}
}

// Test Handle
func (pr *PingRouter) Handler(request biface.IRequest) {
	fmt.Println("Call PingRouter Handle")
	//先读取客户端的数据，再回写ping...ping...ping
	fmt.Println("recv from client : msgID=", request.GetMsgId(), ", data=", string(request.GetData()))

	err := request.GetConn().SendMsg(1, []byte("ping...ping...ping\n"))
	if err != nil {
		fmt.Println("Handle SendMsg err: ", err)
	}
}

// Test PostHandler
func (pr *PingRouter) PostHandler(request biface.IRequest) {
	fmt.Println("Call Router PostHandler")
	err := request.GetConn().SendMsg(1, []byte("After ping .....\n"))
	if err != nil {
		fmt.Println("Post SendMsg err: ", err)
	}
}

type HelloRouter struct {
	BaseRouter
}

func (hr *HelloRouter) Handler(request biface.IRequest) {
	fmt.Println("call helloRouter Handler")
	fmt.Printf("receive from client msgID=%d, data=%s\n", request.GetMsgId(), string(request.GetData()))

	err := request.GetConn().SendMsg(2, []byte("hello btgo hello Router"))
	if err != nil {
		fmt.Println(err)
	}
}

func DoConnectionBegin(conn biface.IConnection) {
	fmt.Println("DoConnectionBegin is Called ... ")
	err := conn.SendMsg(2, []byte("DoConnection BEGIN..."))
	if err != nil {
		fmt.Println(err)
	}
}

func DoConnectionLost(conn biface.IConnection) {
	fmt.Println("DoConnectionLost is Called ... ")
}

func TestServer(t *testing.T) {
	s := NewServer()

	s.SetOnConnStart(DoConnectionBegin)
	s.SetOnConnStop(DoConnectionLost)

	s.AddRouter(1, &PingRouter{})
	s.AddRouter(2, &HelloRouter{})

	go ClientTest(1)
	go ClientTest(2)

	go s.Serve()

	select {
	case <-time.After(time.Second * 10):
		return
	}
}

func TestServerDeadLock(t *testing.T) {
	s := NewServer()

	s.Start()
	time.Sleep(time.Second * 1)

	go func() {
		_, _ = net.Dial("tcp", "127.0.0.1:9091")
	}()
	time.Sleep(time.Second * 1)
	s.Stop()
}

type CloseConnectionBeforeSendMsgRouter struct {
	BaseRouter
}

type DemoPacket struct {
	bpack.DataPack
}

func (d *DemoPacket) Pack(msg biface.IMessage) ([]byte, error) {
	time.Sleep(time.Second * 1)
	return d.DataPack.Pack(msg)
}

func (br *CloseConnectionBeforeSendMsgRouter) Handle(req biface.IRequest) {
	connection := req.GetConn()
	msg := "Btgo server response message for CloseConnectionBeforeSendMsgRouter"
	connection.Stop()
	_ = connection.SendMsg(1, []byte(msg))
	fmt.Println("send: ", msg)
}

func TestCloseConnectionBeforeSendMsg(t *testing.T) {
	s := NewServer()
	s.AddRouter(1, &CloseConnectionBeforeSendMsgRouter{})

	s.Start()
	time.Sleep(time.Second * 1)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		conn, _ := net.Dial("tcp", "127.0.0.1:8999")
		dp := bpack.Factory().NewPack(biface.BtgoDataPack)
		msg := "Btgo client request message for CloseConnectionBeforeSendMsgRouter"
		pack, _ := dp.Pack(bpack.NewMsgPackage(1, []byte(msg)))
		_, _ = conn.Write(pack)
		fmt.Println("send: ", msg)
		buffer := make([]byte, 1024)
		readLen, _ := conn.Read(buffer)
		fmt.Println("received all data: ", string(buffer[dp.GetHeadLen():readLen]))
		wg.Done()
	}()
	wg.Wait()
	s.Stop()
}
