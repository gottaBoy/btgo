package biface

type IRequest interface {
	GetConn() IConnection
	GetData() []byte
}
