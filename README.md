# serial
serial port
## 简介

项目实现了Linux系统多个架构、Windows系统的串行设备的数据读写功能。并能实现对当前建立连接的设备进行再次重新配置参数进行重新运行。

# 例子
```go
func main() {
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
	}
    quit := make(chan os.Signal)
    signal.Notify(quit, os.Interrupt)
    <-quit
}
```