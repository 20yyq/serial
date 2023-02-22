//go:build linux
// +build linux
// @@
// @ Author       : Eacher
// @ Date         : 2022-11-28 09:04:47
// @ LastEditTime : 2023-02-22 08:19:14
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : linux 串口
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /serial/serial_linux.go
// @@
package serial

import (
	"os"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	MIN_TIME 	= 0x01
	MAX_TIME 	= 0xFF

	TCFLSH 		= 0x540B
)

type port struct {
	c 		Config
	f     	*os.File
	t 		*syscall.Termios
}

// 创建一个可用的串口
func New(name string, c Config) (Serial, error) {
	p := &port{t: &syscall.Termios{Iflag: syscall.IGNPAR}}
	var err error
	if p.f, err = os.OpenFile(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666); err != nil {
		return nil, fmt.Errorf("serial new Error: %s", err.Error())
	}
	if err = p.SetConfig(c); err != nil {
		return nil, fmt.Errorf("serial new Error: %s", err.Error())
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, p.f.Fd(), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(p.t))); errno != 0 {
		p.f.Close()
		p.t, p, err = nil, nil, fmt.Errorf("serial restart Error")
	}
	return p, err
}

// 配置端口
func (p *port) SetConfig(c Config) error {
	// 初始化控制模式标志
	p.t.Cflag = 0x00|syscall.CREAD|syscall.CLOCAL

	// 设置波特率
	bauds := map[uint32]uint32{
		50: syscall.B50, 75: syscall.B75, 110: syscall.B110, 134: syscall.B134, 150: syscall.B150, 200: syscall.B200,
		300: syscall.B300, 600: syscall.B600, 1200: syscall.B1200, 1800: syscall.B1800, 2400: syscall.B2400,
		4800: syscall.B4800, 9600: syscall.B9600, 19200: syscall.B19200, 38400: syscall.B38400, 57600: syscall.B57600,
		115200: syscall.B115200, 230400: syscall.B230400, 460800: syscall.B460800, 500000: syscall.B500000,
		576000: syscall.B576000, 921600: syscall.B921600, 1000000: syscall.B1000000, 1152000: syscall.B1152000,
		1500000: syscall.B1500000, 2000000: syscall.B2000000, 2500000: syscall.B2500000, 3000000: syscall.B3000000,
		3500000: syscall.B3500000, 4000000: syscall.B4000000,
	}
	baud, ok := bauds[c.Baud]
	if !ok {
		return fmt.Errorf("unrecognized baud rate")
	}
	p.t.Ispeed, p.t.Ospeed = baud, baud

	// 设置数据位
	switch c.Size {
	case SIZE0:
		fallthrough
	case SIZE8:
		p.t.Cflag |= syscall.CS8
	case SIZE5:
		p.t.Cflag |= syscall.CS5
	case SIZE6:
		p.t.Cflag |= syscall.CS6
	case SIZE7:
		p.t.Cflag |= syscall.CS7
	default:
		return fmt.Errorf("unsupported serial data size")
	}

	// 设置停止位
	switch c.StopBits {
	case STOP0:
	case STOP1:
	case STOP2:
		p.t.Cflag |= syscall.CSTOPB
	default:
		return fmt.Errorf("unsupported stop bit setting")
	}

	// 设置校验位
	switch c.Parity {
	case PARITY_ZERO:
	case PARITY_NONE:
	case PARITY_ODD:
		p.t.Cflag |= syscall.PARENB
		p.t.Cflag |= syscall.PARODD
	case PARITY_EVEN:
		p.t.Cflag |= syscall.PARENB
	default:
		return fmt.Errorf("unsupported parity setting")
	}

	/**	VTIME和VMIN值,这两个值只用于非标准模式,两者结合共同控制对输入的读取方式,还能控制在一个程序试图与一个终端关联的文件描述符时将发生的情况
	 *  
	 *  VMIN = 0, VTIME = 0时:Read方法立即返回,如果有待处理的字符,它们就会被返回,如果没有,Read方法调用返回0,且不读取任何字符
	 *  
	 *  VMIN = 0, VTIME > 0时:有字符处理或经过VTIME个0.1秒后返回
	 *  
	 *  VMIN > 0, VTIME = 0时:Read方法一直等待,直到有VMIN个字符可以读取,返回值是字符的数量.到达文件尾时返回0
	 *  
	 *  VMIN > 0, VTIME > 0时:Read方法调用时,它会等待接收一个字符.在接收到第一个字符及其后续的每个字符后,启用一个字符间隔定时器.
	 *  当有VMIN个字符可读或两字符间的时间间隔超进VTIME个0.1秒时,Read方法返回
	**/
	ms := c.ReadTime.Milliseconds()/100
	if ms < MIN_TIME {
		ms = MIN_TIME
	} else if ms > MAX_TIME {
		ms = MAX_TIME
	}
	p.t.Cc[syscall.VMIN], p.t.Cc[syscall.VTIME] = c.MinByte, uint8(ms)
	return nil
}

func (p *port) Read(b []byte) (n int, err error) {
	if n, err = p.f.Read(b); err != nil {
		return 0, fmt.Errorf("serial read Error: %s", err.Error())
	}
	return n, nil
}

func (p *port) Write(b []byte) (n int, err error) {
	if n, err = p.f.Write(b); err != nil {
		return 0, fmt.Errorf("serial write Error: %s", err.Error())
	}
	return n, nil
}

func (p *port) InFlush() error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, p.f.Fd(), uintptr(TCFLSH), uintptr(syscall.TCIFLUSH)); errno != 0 {
		return fmt.Errorf("serial InFlush Error")
	}
	return nil
}

func (p *port) OutFlush() error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, p.f.Fd(), uintptr(TCFLSH), uintptr(syscall.TCOFLUSH)); errno != 0 {
		return fmt.Errorf("serial InFlush Error")
	}
	return nil
}

func (p *port) Close() (err error) {
	if err = p.f.Close(); err != nil {
		switch er := err.(type) {
		case *os.PathError:
			if er.Err.Error() == os.ErrClosed.Error() {
				return nil
			}
		default:
		}
		return fmt.Errorf("serial close Error: %s", err.Error())
	}
	return nil
}

func (p *port) RestStart() error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, p.f.Fd(), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(p.t))); errno != 0 {
		return fmt.Errorf("serial restart Error")
	}
	return nil
}
