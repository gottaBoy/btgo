package biface

type IRequest interface {
	GetConn() IConnection
	GetData() []byte
	GetMsgId() uint32
}
