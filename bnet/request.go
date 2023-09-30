package bnet

import (
	"btgo/biface"
)

type Request struct {
	conn biface.IConnection
	data []byte
}

func (r *Request) GetConn() biface.IConnection {
	return r.conn
}

func (r *Request) GetData() []byte {
	return r.data
}
