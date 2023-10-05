package utils

import (
	"btgo/biface"
	"encoding/json"
	"io/ioutil"
)

type GlobalObj struct {
	Server         biface.IServer
	Host           string
	Name           string
	Port           int
	Version        string
	MaxPacketSize  uint32
	MaxConnSize    uint32
	WorkerPoolSize uint32
	MaxWorkerTask  uint32
	ConfFilePath   string
}

var GlobalObject *GlobalObj

func init() {
	GlobalObject = &GlobalObj{
		Name:           "BTGO server",
		Host:           "0.0.0.0",
		Port:           7777,
		Version:        "v1.0.0",
		MaxPacketSize:  4096,
		MaxConnSize:    10000,
		WorkerPoolSize: 10,
		MaxWorkerTask:  1024,
		ConfFilePath:   "conf/zinx.json",
	}
	GlobalObject.Reload()
}

func (obj *GlobalObj) Reload() {
	data, err := ioutil.ReadFile("conf/btgo.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &GlobalObject)
	if err != nil {
		panic(err)
	}
}
