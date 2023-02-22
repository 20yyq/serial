// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 14:42:42
// @ LastEditTime : 2023-02-22 08:27:33
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /serial/examples/main.go
// @@
package main

import (
    "os"
    "time"
    "fmt"
    "os/signal"

	"github.com/20yyq/serial"
)

func main() {
	// c := serial.Config{Baud: 115200, MinByte: 10, ReadTime: time.Duration(2500000)}
	// conn, err := serial.New("/dev/ttyACM0", c)
	c := serial.Config{Baud: 115200, ReadTime: time.Hour}
	conn, err := serial.New("COM9", c)
	if err != nil {
		fmt.Println("conn new error:", err)
		return
	}
	b := make([]byte, 1024)
	for {
		n, err := conn.Read(b)
		if err != nil {
			fmt.Println("conn.Read error:", err)
			break
		}
		fmt.Println("Read byte:", b[:n])
		// fmt.Println(conn.RestStart())
		// fmt.Println("Read byte:", b[:n], conn.InFlush(), conn.OutFlush())
	}
    quit := make(chan os.Signal)
    signal.Notify(quit, os.Interrupt)
    <-quit
}