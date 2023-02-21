//go:build windows
// +build windows
// @@
// @ Author       : Eacher
// @ Date         : 2023-02-21 09:46:27
// @ LastEditTime : 2023-02-21 15:51:20
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : win 串行设备
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /goserial/serials/serial_windows.go
// @@
package serials

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

const (
	MIN_TIME 	= 0x01
	MAX_TIME 	= 0xFFFFFFFF

	EV_RXCHAR     = 0x0001
	PURGE_TXABORT = 0x0001
	PURGE_RXABORT = 0x0002
	PURGE_TXCLEAR = 0x0004
	PURGE_RXCLEAR = 0x0008
)

var (
	onec sync.Once

	k32DLL *syscall.DLL

	setCommState, getCommState, setCommTimeouts, setCommMask, setupComm, purgeComm uintptr
)

type port struct {
	h  syscall.Handle
	rl sync.Mutex
	wl sync.Mutex
	dcb _DCB
}

type _DCB struct {
	DCBlength 	uint32
	BaudRate 	uint32
	flags 		[4]byte
	wReserved 	uint16
	XonLim 		uint16
	XoffLim 	uint16
	ByteSize 	byte
	Parity 		byte
	StopBits 	byte
	XonChar 	byte
	XoffChar 	byte
	ErrorChar 	byte
	EofChar 	byte
	EvtChar 	byte
	wReserved1 	uint16
}

type _COMMTIMEOUTS struct {
	ReadIntervalTimeout 		uint32
	ReadTotalTimeoutMultiplier 	uint32
	ReadTotalTimeoutConstant 	uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant 	uint32
}

// 创建一个可用的串口
func New(name string, c Config) (Serial, error) {
	var err error
	onec.Do(func() { err = initK32DLL() })
	if err != nil {
		return nil, fmt.Errorf("k32DLL load Error: %s", err.Error())
	}
	if len(name) > 0 && name[0] != '\\' {
		name = `\\.\` + name
	}
	p := &port{}
	if p.h, err = syscall.CreateFile(syscall.StringToUTF16Ptr(name),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL, 0); err != nil {
		return nil, fmt.Errorf("serial new Error: %s", err.Error())
	}
	p.dcb.DCBlength = uint32(unsafe.Sizeof(p.dcb))
	if r, _, _ := syscall.Syscall(getCommState, 2, uintptr(p.h), uintptr(unsafe.Pointer(&p.dcb)), 0); r == 0 {
		return nil, fmt.Errorf("serial new Error: %s", syscall.GetLastError())
	}
	if err = p.SetConfig(c); err != nil {
		return nil, fmt.Errorf("serial new Error: %s", err.Error())
	}
	if err = p.initComm(); err != nil {
		return nil, fmt.Errorf("serial initComm Error: %s", err.Error())
	}
	return p, nil
}

// 配置端口
func (p *port) SetConfig(c Config) error {
	// 设置波特率
	p.dcb.BaudRate = c.Baud

	// 设置数据位
	p.dcb.ByteSize = c.Size
	if p.dcb.ByteSize < 5 && p.dcb.ByteSize > 8 {
		p.dcb.ByteSize = SIZE8
	}

	// 设置停止位
	switch c.StopBits {
	case STOP0:
		fallthrough
	case STOPHALF:
		p.dcb.StopBits = STOP1
	case STOP1:
		p.dcb.StopBits = STOP0
	case STOP2:
		p.dcb.StopBits = STOP2
	default:
		return fmt.Errorf("unsupported stop bit setting")
	}

	// 设置校验位
	switch c.Parity {
	case PARITY_ZERO:
		fallthrough
	case PARITY_NONE:
		p.dcb.Parity = 0
	case PARITY_ODD:
		p.dcb.Parity = 1
	case PARITY_EVEN:
		p.dcb.Parity = 2
	case PARITY_MARK:
		p.dcb.Parity = 3
	case PARITY_SPACE:
		p.dcb.Parity = 4
	default:
		return fmt.Errorf("unsupported parity setting")
	}

	// 超时设置
	ms := c.ReadTime.Milliseconds()
	if ms < 1 {
		ms = MIN_TIME
	} else if ms > MAX_TIME {
		ms = MAX_TIME
	}
	timeouts := &_COMMTIMEOUTS{
		ReadIntervalTimeout:        MAX_TIME,
		ReadTotalTimeoutMultiplier: MAX_TIME,
		ReadTotalTimeoutConstant:   uint32(ms),
	}
	if _, _, errno := syscall.Syscall(setCommTimeouts, 2, uintptr(p.h), uintptr(unsafe.Pointer(timeouts)), 0); errno == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func (p *port) Read(buf []byte) (int, error) {
	p.rl.Lock()
	defer p.rl.Unlock()
	n, err := syscall.Read(p.h, buf)
	if err != nil {
		if err.(syscall.Errno) == syscall.ERROR_IO_PENDING {
			err = nil
		}
	}
	return n, err
}

func (p *port) Write(buf []byte) (int, error) {
	p.wl.Lock()
	defer p.wl.Unlock()
	n, err := syscall.Write(p.h, buf)
	if err != nil {
		if err.(syscall.Errno) == syscall.ERROR_IO_PENDING {
			err = nil
		}
	}
	return n, err
}

func (p *port) InFlush() error {
	if _, _, errno := syscall.Syscall(purgeComm, 2, uintptr(p.h), PURGE_RXCLEAR|PURGE_RXABORT, 0); errno == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func (p *port) OutFlush() error {
	if _, _, errno := syscall.Syscall(purgeComm, 2, uintptr(p.h), PURGE_TXCLEAR|PURGE_TXABORT, 0); errno == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func (p *port) Close() error {
	return syscall.Close(p.h)
}

func (p *port) RestStart() error {
	p.dcb.DCBlength = uint32(unsafe.Sizeof(p.dcb))
	if _, _, errno := syscall.Syscall(setCommState, 2, uintptr(p.h), uintptr(unsafe.Pointer(&p.dcb)), 0); errno == 0 {
		return fmt.Errorf("serial RestStart Error: %s", syscall.GetLastError())
	}
	return nil
}

func (p *port) initComm() error {
	p.dcb.DCBlength = uint32(unsafe.Sizeof(p.dcb))
	if _, _, errno := syscall.Syscall(setCommState, 2, uintptr(p.h), uintptr(unsafe.Pointer(&p.dcb)), 0); errno == 0 {
		return syscall.GetLastError()
	}
	if _, _, errno := syscall.Syscall(setupComm, 3, uintptr(p.h), 512, 512); errno == 0 {
		return syscall.GetLastError()
	}
	if _, _, errno := syscall.Syscall(setCommMask, 2, uintptr(p.h), EV_RXCHAR, 0); errno == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func initK32DLL() error {
	var err error
	k32DLL, err = syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	defer syscall.FreeLibrary(k32DLL.Handle)
	if setupComm, err = syscall.GetProcAddress(k32DLL.Handle, "SetupComm"); err != nil {
		return err
	}
	if purgeComm, err = syscall.GetProcAddress(k32DLL.Handle, "PurgeComm"); err != nil {
		return err
	}
	if setCommMask, err = syscall.GetProcAddress(k32DLL.Handle, "SetCommMask"); err != nil {
		return err
	}
	if getCommState, err = syscall.GetProcAddress(k32DLL.Handle, "GetCommState"); err != nil {
		return err
	}
	if setCommState, err = syscall.GetProcAddress(k32DLL.Handle, "SetCommState"); err != nil {
		return err
	}
	setCommTimeouts, err = syscall.GetProcAddress(k32DLL.Handle, "SetCommTimeouts")
	return err
}
