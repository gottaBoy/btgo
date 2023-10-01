package bnet

import (
	"btgo/biface"
)

type Request struct {
	conn biface.IConnection
	// data []byte
	msg biface.IMessage
}

func (r *Request) GetConn() biface.IConnection {
	return r.conn
}

func (r *Request) GetData() []byte {
	return r.msg.GetData()
}

func (r *Request) GetMsgId() uint32 {
	return r.msg.GetMsgId()
}
