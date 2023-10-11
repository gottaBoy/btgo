package logo

import (
	"btgo/bconf"
	"fmt"
)

var btLogo = `
BTGO
`
var topLine = `┌──────────────────────────────────────────────────────┐`
var borderLine = `│`
var bottomLine = `└──────────────────────────────────────────────────────┘`

func PrintLogo() {
	fmt.Println(btLogo)
	fmt.Println(topLine)
	fmt.Println(fmt.Sprintf("%s [Github] https://github.com/gottaboy                    %s", borderLine, borderLine))
	fmt.Println(fmt.Sprintf("%s [tutorial] https://www.yuque.com/gottaboy %s", borderLine, borderLine))
	fmt.Println(fmt.Sprintf("%s [document] https://www.yuque.com/gottaboy      %s", borderLine, borderLine))
	fmt.Println(bottomLine)
	fmt.Printf("[Btgo] Version: %s, MaxConn: %d, MaxPacketSize: %d\n",
		bconf.GlobalObject.Version,
		bconf.GlobalObject.MaxConnSize,
		bconf.GlobalObject.MaxPacketSize)
}
