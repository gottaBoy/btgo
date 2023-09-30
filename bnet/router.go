package bnet

import (
	"btgo/biface"
)

type BaseRouter struct{}

func (br *BaseRouter) PreHandler(request biface.IRequest)  {}
func (br *BaseRouter) Handler(request biface.IRequest)     {}
func (br *BaseRouter) PostHandler(request biface.IRequest) {}
