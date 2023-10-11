package binterceptor

import "btgo/biface"

type Chain struct {
	req          biface.IcReq
	position     int
	interceptors []biface.IInterceptor
}

func NewChain(list []biface.IInterceptor, pos int, req biface.IcReq) biface.IChain {
	return &Chain{
		req:          req,
		position:     pos,
		interceptors: list,
	}
}

func (c *Chain) Request() biface.IcReq {
	return c.req
}

func (c *Chain) Proceed(request biface.IcReq) biface.IcResp {
	if c.position < len(c.interceptors) {
		chain := NewChain(c.interceptors, c.position+1, request)
		interceptor := c.interceptors[c.position]
		response := interceptor.Intercept(chain)
		return response
	}
	return request
}

// GetIMessage  从Chain中获取IMessage
func (c *Chain) GetIMessage() biface.IMessage {

	req := c.Request()
	if req == nil {
		return nil
	}

	iRequest := c.ShouldIRequest(req)
	if iRequest == nil {
		return nil
	}

	return iRequest.GetMessage()
}

// Next 通过IMessage和解码后数据进入下一个责任链任务
// iMessage 为解码后的IMessage
// response 为解码后的数据
func (c *Chain) ProceedWithIMessage(iMessage biface.IMessage, response biface.IcReq) biface.IcResp {
	if iMessage == nil || response == nil {
		return c.Proceed(c.Request())
	}

	req := c.Request()
	if req == nil {
		return c.Proceed(c.Request())
	}

	iRequest := c.ShouldIRequest(req)
	if iRequest == nil {
		return c.Proceed(c.Request())
	}

	//设置chain的request下一次请求
	iRequest.SetResponse(response)

	return c.Proceed(iRequest)
}

// ShouldIRequest 判断是否是IRequest
func (c *Chain) ShouldIRequest(icReq biface.IcReq) biface.IRequest {
	if icReq == nil {
		return nil
	}

	switch icReq.(type) {
	case biface.IRequest:
		return icReq.(biface.IRequest)
	default:
		return nil
	}
}
