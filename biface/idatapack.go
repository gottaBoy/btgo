package biface

type IDataPack interface {
	GetHeadLen() uint32
	Pack(msg IMessage) ([]byte, error)
	UnPack([]byte) (IMessage, error)
}

const (
	// Btgo standard packing and unpacking method (Btgo 标准封包和拆包方式)
	BtgoDataPack    string = "btgo_pack_tlv_big_endian"
	BtgoDataPackOld string = "btgo_pack_ltv_little_endian"

	//...(+)
	//// Custom packing method can be added here(自定义封包方式在此添加)
)

const (
	// Btgo default standard message protocol format(Btgo 默认标准报文协议格式)
	BtgoMessage string = "btgo_message"
)
