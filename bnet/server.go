package bnet

import (
	"btgo/biface"
	"btgo/utils"
	"errors"
	"fmt"
	"net"
	"time"
)

type Server struct {
	Name    string
	Network string
	IP      string
	Port    int
	// Router  biface.IRouter
	MsgHandler biface.IMsgHandler
}

func CallBackHandler(conn *net.TCPConn, data []byte, cnt int) error {
	fmt.Println("[Conn Handler] CallBackHandler ... ")
	if _, err := conn.Write(data[:cnt]); err != nil {
		fmt.Println("write back buf err ", err)
		return errors.New("CallBackHandler error")
	}
	return nil
}

func (s *Server) AddRouter(msgId uint32, router biface.IRouter) {
	// s.Router = router
	s.MsgHandler.AddRouter(msgId, router)
}

func (s *Server) Start() {
	fmt.Printf("[START] Server name: %s,listenner at IP: %s, Port %d is starting\n", s.Name, s.IP, s.Port)
	fmt.Printf("[BTGO] Version: %s, MaxConnSize: %d,  MaxPacketSize: %d\n",
		utils.GlobalObject.Version,
		utils.GlobalObject.MaxConnSize,
		utils.GlobalObject.MaxPacketSize)

	go func() {
		fmt.Println("获取链接成功")
		addr, err := net.ResolveTCPAddr(s.Network, fmt.Sprintf("%s:%d", s.IP, s.Port))
		if err != nil {
			fmt.Println("Resolve addr err: ", err)
			return
		}
		fmt.Println("获取链接成功")
		// 获取监听器
		listenner, err := net.ListenTCP(s.Network, addr)
		if err != nil {
			fmt.Println("Listener ", s.Network, "err", err)
			return
		}

		// 监听器接收请求
		for {
			conn, err := listenner.AcceptTCP()
			if err != nil {
				fmt.Println("Accept err ", err)
				continue
			}
			fmt.Println("获取链接成功")
			//TODO 设置最大连接
			var connId uint32
			connId = 0
			newConn := NewConn(conn, connId, s.MsgHandler)
			connId++
			// 启动处理任务
			go newConn.Open()
		}
	}()
}

func (s *Server) Stop() {
	fmt.Println("[STOP] btgo server , name ", s.Name)

	for {
		time.Sleep(10 * time.Second)
	}
}

func (s *Server) Serve() {
	fmt.Println("[Server] btgo server , name ", s.Name)
	s.Start()
	for {
		time.Sleep(10 * time.Second)
	}
}

func NewServer(name string) biface.IServer {
	utils.GlobalObject.Reload()

	s := &Server{
		Name:    utils.GlobalObject.Name,
		Network: "tcp4",
		IP:      utils.GlobalObject.Host,
		Port:    utils.GlobalObject.Port,
		// Router:  nil,
		MsgHandler: NewMsgHandler(),
	}
	return s
}
