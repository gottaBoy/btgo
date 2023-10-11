package bnet

import (
	"fmt"
	"testing"

	"btgo/biface"
)

func A1(request biface.IRequest) {
	fmt.Println("test A1")
}
func A2(request biface.IRequest) {
	fmt.Println("test A2")
}
func A3(request biface.IRequest) {
	fmt.Println("test A3")
}
func A4(request biface.IRequest) {
	fmt.Println("test A4")
}
func A5(request biface.IRequest) {
	fmt.Println("test A5")
}

func TestRouterAdd(t *testing.T) {
	router := NewRouterSlices()
	router.Use(A3)
	router.AddHandler(1, A1, A2)

	testgroup := router.Group(2, 5, A5)
	{
		testgroup.AddHandler(2, A4)

		//正确panic
		//testgroup.AddHandler(6, A4)
	}

	for _, v := range router.Routers[2] {
		v(&Request{
			index: -1,
		})
	}
}
