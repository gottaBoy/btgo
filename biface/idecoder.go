package biface

type IDecoder interface {
	IInterceptor
	GetLengthField() *LengthField
}
