package bconf

import (
	"btgo/biface"
	"btgo/blogger"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

/*
global config
*/
const (
	ServerModeTcp       = "tcp"
	ServerModeWebsocket = "websocket"
	ServerModeKcp       = "kcp"
)

const (
	WorkerModeHash = "Hash" // By default, the round-robin average allocation rule is used.(默认使用取余的方式)
	WorkerModeBind = "Bind" // Bind a worker to each connection.(为每个连接分配一个worker)
)

type GlobalObj struct {
	Server              biface.IServer
	Host                string
	Name                string
	TCPPort             int
	Version             string
	MaxPacketSize       uint32
	MaxConnSize         uint32
	WorkerPoolSize      uint32
	MaxWorkerTaskLenLen uint32
	MaxMsgChanLen       uint32
	ConfFilePath        string
}

/*
	   Store all global parameters related to the Btgo framework for use by other modules.
	   Some parameters can also be configured by the user based on the Btgo.json file.
		(存储一切有关Btgo框架的全局参数，供其他模块使用
		一些参数也可以通过 用户根据 Btgo.json来配置)
*/
type Config struct {
	/*
		Server
	*/
	Host    string // The IP address of the current server. (当前服务器主机IP)
	TCPPort int    // The port number on which the server listens for TCP connections.(当前服务器主机监听端口号)
	WsPort  int    // The port number on which the server listens for WebSocket connections.(当前服务器主机websocket监听端口)
	Name    string // The name of the current server.(当前服务器名称)
	KcpPort int    // he port number on which the server listens for KCP connections.(当前服务器主机监听端口号)

	/*
		Btgo
	*/
	Version          string // The version of the Btgo framework.(当前Btgo版本号)
	MaxPacketSize    uint32 // The maximum size of the packets that can be sent or received.(读写数据包的最大值)
	MaxConnSize      int    // The maximum number of connections that the server can handle.(当前服务器主机允许的最大链接个数)
	WorkerPoolSize   uint32 // The number of worker pools in the business logic.(业务工作Worker池的数量)
	MaxWorkerTaskLen uint32 // The maximum number of tasks that a worker pool can handle.(业务工作Worker对应负责的任务队列最大任务存储数量)
	WorkerMode       string // The way to assign workers to connections.(为链接分配worker的方式)
	MaxMsgChanLen    uint32 // The maximum length of the send buffer message queue.(SendBuffMsg发送消息的缓冲最大长度)
	IOReadBuffSize   uint32 // The maximum size of the read buffer for each IO operation.(每次IO最大的读取长度)

	//The server mode, which can be "tcp" or "websocket". If it is empty, both modes are enabled.
	//"tcp":tcp监听, "websocket":websocket 监听 为空时同时开启
	Mode string

	// A boolean value that indicates whether the new or old version of the router is used. The default value is false.
	// 路由模式 false为旧版本路由，true为启用新版本的路由 默认使用旧版本
	RouterSlicesMode bool

	/*
		logger
	*/
	LogDir string // The directory where log files are stored. The default value is "./log".(日志所在文件夹 默认"./log")

	// The name of the log file. If it is empty, the log information will be printed to stderr.
	// (日志文件名称   默认""  --如果没有设置日志文件，打印信息将打印至stderr)
	LogFile string

	LogSaveDays int   // 日志最大保留天数
	LogFileSize int64 // 日志单个日志最大容量 默认 64MB,单位：字节，记得一定要换算成MB（1024 * 1024）
	LogCons     bool  // 日志标准输出  默认 false

	// The level of log isolation. The values can be 0 (all open), 1 (debug off), 2 (debug/info off), 3 (debug/info/warn off), and so on.
	// 日志隔离级别  -- 0：全开 1：关debug 2：关debug/info 3：关debug/info/warn ...
	LogIsolationLevel int

	/*
		Keepalive
	*/
	// The maximum interval for heartbeat detection in seconds.
	// 最长心跳检测间隔时间(单位：秒),超过改时间间隔，则认为超时，从配置文件读取
	HeartbeatMax int

	/*
		TLS
	*/
	CertFile       string // The name of the certificate file. If it is empty, TLS encryption is not enabled.(证书文件名称 默认"")
	PrivateKeyFile string // The name of the private key file. If it is empty, TLS encryption is not enabled.(私钥文件名称 默认"" --如果没有设置证书和私钥文件，则不启用TLS加密)
}

var GlobalObject *Config

// func init() {
// 	GlobalObject = &GlobalObj{
// 		Name:                "BTGO server",
// 		Host:                "0.0.0.0",
// 		Port:                7777,
// 		Version:             "v1.0.0",
// 		MaxPacketSize:       4096,
// 		MaxConnSize:         10000,
// 		WorkerPoolSize:      10,
// 		MaxWorkerTaskLenLen: 1024,
// 		MaxMsgChanLen:       10,
// 		ConfFilePath:        "conf/Btgo.json",
// 	}
// 	GlobalObject.Reload()
// }

// func (obj *GlobalObj) Reload() {
// 	confFilePath := GetConfigFilePath()
// 	if confFileExists, _ := PathExists(confFilePath); confFileExists != true {

// 		// The configuration file may not exist,
// 		// in which case the default parameters should be used to initialize the logging module configuration.
// 		// (配置文件不存在也需要用默认参数初始化日志模块配置)
// 		g.InitLogConfig()

// 		blogger.Ins().ErrorF("Config File %s is not exist!! \n You can set configFile by setting the environment variable %s, like export %s = xxx/xxx/Btgo.conf ", confFilePath, EnvConfigFilePathKey, EnvConfigFilePathKey)
// 		return
// 	}

// 	data, err := os.ReadFile(confFilePath)
// 	// data, err := ioutil.ReadFile("conf/btgo.json")
// 	if err != nil {
// 		panic(err)
// 	}
// 	err = json.Unmarshal(data, &GlobalObject)
// 	if err != nil {
// 		panic(err)
// 	}
// 	g.InitLogConfig()
// }

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (g *Config) Reload() {
	confFilePath := GetConfigFilePath()
	if confFileExists, _ := PathExists(confFilePath); confFileExists != true {

		// The configuration file may not exist,
		// in which case the default parameters should be used to initialize the logging module configuration.
		// (配置文件不存在也需要用默认参数初始化日志模块配置)
		g.InitLogConfig()

		blogger.Ins().ErrorF("Config File %s is not exist!! \n You can set configFile by setting the environment variable %s, like export %s = xxx/xxx/Btgo.conf ", confFilePath, EnvConfigFilePathKey, EnvConfigFilePathKey)
		return
	}

	data, err := os.ReadFile(confFilePath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, g)
	if err != nil {
		panic(err)
	}

	g.InitLogConfig()
}

// Show Btgo Config Info
func (g *Config) Show() {
	objVal := reflect.ValueOf(g).Elem()
	objType := reflect.TypeOf(*g)

	fmt.Println("===== Btgo Global Config =====")
	for i := 0; i < objVal.NumField(); i++ {
		field := objVal.Field(i)
		typeField := objType.Field(i)

		fmt.Printf("%s: %v\n", typeField.Name, field.Interface())
	}
	fmt.Println("==============================")
}

func (g *Config) HeartbeatMaxDuration() time.Duration {
	return time.Duration(g.HeartbeatMax) * time.Second
}

func (g *Config) InitLogConfig() {
	if g.LogFile != "" {
		blogger.SetLogFile(g.LogDir, g.LogFile)
		blogger.SetCons(g.LogCons)
	}
	if g.LogSaveDays > 0 {
		blogger.SetMaxAge(g.LogSaveDays)
	}
	if g.LogFileSize > 0 {
		blogger.SetMaxSize(g.LogFileSize)
	}
	if g.LogIsolationLevel > blogger.LogDebug {
		blogger.SetLogLevel(g.LogIsolationLevel)
	}
}

/*
init, set default value
*/
func init() {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "."
	}

	// Note: Prevent errors like "flag provided but not defined: -test.paniconexit0" from occurring in go test.
	// (防止 go test 出现"flag provided but not defined: -test.paniconexit0"等错误)
	testing.Init()

	// Initialize the GlobalObject variable and set some default values.
	// (初始化GlobalObject变量，设置一些默认值)
	GlobalObject = &Config{
		Name:              "BtgoServerApp",
		Version:           "V1.0",
		TCPPort:           9090,
		WsPort:            9000,
		KcpPort:           9001,
		Host:              "0.0.0.0",
		MaxConnSize:       12000,
		MaxPacketSize:     4096,
		WorkerPoolSize:    10,
		MaxWorkerTaskLen:  1024,
		WorkerMode:        "",
		MaxMsgChanLen:     1024,
		LogDir:            pwd + "/log",
		LogFile:           "", // if set "", print to Stderr(默认日志文件为空，打印到stderr)
		LogIsolationLevel: 0,
		HeartbeatMax:      10, // The default maximum interval for heartbeat detection is 10 seconds. (默认心跳检测最长间隔为10秒)
		IOReadBuffSize:    1024,
		CertFile:          "",
		PrivateKeyFile:    "",
		Mode:              ServerModeTcp,
		RouterSlicesMode:  false,
	}

	// Note: Load some user-configured parameters from the configuration file.
	// (从配置文件中加载一些用户配置的参数)
	GlobalObject.Reload()
}
