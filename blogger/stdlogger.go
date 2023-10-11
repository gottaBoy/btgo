package blogger

/*
	A global Log handle is provided by default for external use, which can be called directly through the API series.
	The global log object is StdLogger.
	Note: The methods in this file do not support customization and cannot replace the log recording mode.

	If you need a custom logger, please use the following methods:
	blogger.SetLogger(yourLogger)
	blogger.Ins().InfoF() and other methods.

   全局默认提供一个Log对外句柄，可以直接使用API系列调用
   全局日志对象 StdLogger
   注意：本文件方法不支持自定义，无法替换日志记录模式，如果需要自定义Logger:

   请使用如下方法:
   blogger.SetLogger(yourLogger)
   blogger.Ins().InfoF()等方法
*/

// StdLogger creates a global log
var StdLogger = NewLogger("", BitDefault)

// Flags gets the flags of StdLogger
func Flags() int {
	return StdLogger.Flags()
}

// ResetFlags sets the flags of StdLogger
func ResetFlags(flag int) {
	StdLogger.ResetFlags(flag)
}

// AddFlag adds a flag to StdLogger
func AddFlag(flag int) {
	StdLogger.AddFlag(flag)
}

// SetPrefix sets the log prefix of StdLogger
func SetPrefix(prefix string) {
	StdLogger.SetPrefix(prefix)
}

// SetLogFile sets the log file of StdLogger
func SetLogFile(fileDir string, fileName string) {
	StdLogger.SetLogFile(fileDir, fileName)
}

// SetMaxAge 最大保留天数
func SetMaxAge(ma int) {
	StdLogger.SetMaxAge(ma)
}

// SetMaxSize 单个日志最大容量 单位：字节
func SetMaxSize(ms int64) {
	StdLogger.SetMaxSize(ms)
}

// SetCons 同时输出控制台
func SetCons(b bool) {
	StdLogger.SetCons(b)
}

// SetLogLevel sets the log level of StdLogger
func SetLogLevel(logLevel int) {
	StdLogger.SetLogLevel(logLevel)
}

func Debugf(format string, v ...interface{}) {
	StdLogger.Debugf(format, v...)
}

func Debug(v ...interface{}) {
	StdLogger.Debug(v...)
}

func Infof(format string, v ...interface{}) {
	StdLogger.Infof(format, v...)
}

func Info(v ...interface{}) {
	StdLogger.Info(v...)
}

func Warnf(format string, v ...interface{}) {
	StdLogger.Warnf(format, v...)
}

func Warn(v ...interface{}) {
	StdLogger.Warn(v...)
}

func Errorf(format string, v ...interface{}) {
	StdLogger.Errorf(format, v...)
}

func Error(v ...interface{}) {
	StdLogger.Error(v...)
}

func Fatalf(format string, v ...interface{}) {
	StdLogger.Fatalf(format, v...)
}

func Fatal(v ...interface{}) {
	StdLogger.Fatal(v...)
}

func Panicf(format string, v ...interface{}) {
	StdLogger.Panicf(format, v...)
}

func Panic(v ...interface{}) {
	StdLogger.Panic(v...)
}

func Stack(v ...interface{}) {
	StdLogger.Stack(v...)
}

func init() {
	// Since the StdLogger object wraps all output methods with an extra layer, the call depth is one more than a normal logger object
	// The call depth of a regular Loggerger object is 2, and the call depth of StdLogger is 3
	// (因为StdLogger对象 对所有输出方法做了一层包裹，所以在打印调用函数的时候，比正常的logger对象多一层调用
	// 一般的Loggerger对象 calldDepth=2, StdLogger的calldDepth=3)
	StdLogger.calldDepth = 3
}
