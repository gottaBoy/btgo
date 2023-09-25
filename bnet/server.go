package bnet

import (
	"btgo/biface"
	"fmt"
	"net"
	"time"
)

type Server struct {
	Name    string
	Network string
	IP      string
	Port    int
}

func (s *Server) Start() {
	fmt.Printf("IP: %s, Port: %d", s.IP, s.Port)
	go func() {
		addr, err := net.ResolveTCPAddr(s.Network, fmt.Sprintf("%s:%d", s.IP, s.Port))
		if err != nil {
			fmt.Println("Resolve addr err: ", err)
			return
		}

		listenner, err := net.ListenTCP(s.Network, addr)
		if err != nil {
			fmt.Println("Listener ", s.Network, "err", err)
			return
		}

		for {
			conn, err := listenner.AcceptTCP()
			if err != nil {
				fmt.Println("Accept err ", err)
				continue
			}

			go func() {
				for {
					buf := make([]byte, 512)
					cnt, err := conn.Read(buf)
					if err != nil {
						fmt.Println("Recv buf err ", err)
						continue
					}
					// 回显
					if _, err := conn.Write(buf[:cnt]); err != nil {
						fmt.Println("Write back buf err ", err)
						continue
					}
				}
			}()
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
}

func NewServer(name string) biface.IServer {
	s := &Server{
		Name:    name,
		Network: "tcp4",
		IP:      "0.0.0.0",
		Port:    7777,
	}
	return s
}
