package biface

import "net"

type IConnection interface {
	Open()
	Close()
	IsClose() bool
	GetConn() *net.TCPConn
	GetConnId() uint32
	GetAdrr() net.Addr
}

// 统一处理函数
type HandlerFunc func(*net.TCPConn, []byte, int) error
