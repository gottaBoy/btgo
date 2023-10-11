package blogger

import (
	"btgo/biface"
	"context"
	"fmt"
)

var LogInstance biface.ILogger = new(DefaultLog)

type DefaultLog struct{}

func (log *DefaultLog) InfoF(format string, v ...interface{}) {
	StdLogger.Infof(format, v...)
}

func (log *DefaultLog) ErrorF(format string, v ...interface{}) {
	StdLogger.Errorf(format, v...)
}

func (log *DefaultLog) DebugF(format string, v ...interface{}) {
	StdLogger.Debugf(format, v...)
}

func (log *DefaultLog) InfoFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdLogger.Infof(format, v...)
}

func (log *DefaultLog) ErrorFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdLogger.Errorf(format, v...)
}

func (log *DefaultLog) DebugFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdLogger.Debugf(format, v...)
}

func SetLogger(newlog biface.ILogger) {
	LogInstance = newlog
}

func Ins() biface.ILogger {
	return LogInstance
}
