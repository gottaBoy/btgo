package biface

import "net"

type IConnection interface {
	Open()
	Close()
	IsClose() bool
	GetConn() *net.TCPConn
	GetConnId() uint32
	GetAdrr() net.Addr
	SendMsg(msgId uint32, msg []byte) error
}

// 统一处理函数
type HandlerFunc func(*net.TCPConn, []byte, int) error
