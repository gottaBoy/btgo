package bnet

import "btgo/biface"

type RequestFunc struct {
	biface.BaseRequest
	conn     biface.IConnection
	callFunc func()
}

func (rf *RequestFunc) GetConn() biface.IConnection {
	return rf.conn
}

func (rf *RequestFunc) CallFunc() {
	if rf.callFunc != nil {
		rf.callFunc()
	}
}

func NewFuncRequest(conn biface.IConnection, callFunc func()) biface.IRequest {
	req := new(RequestFunc)
	req.conn = conn
	req.callFunc = callFunc
	return req
}
