// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 13:45:45
// @ LastEditTime : 2023-12-25 16:29:27
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  :
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /20yyq/serial/serial.go
// @@
package serial

import (
	"time"
)

const (
	SIZE0 = 0
	SIZE5 = 5
	SIZE6 = 6
	SIZE7 = 7
	SIZE8 = 8

	STOP0    = 0
	STOP1    = 1
	STOP2    = 2
	STOPHALF = 15

	PARITY_ZERO  = 0
	PARITY_NONE  = 'N'
	PARITY_ODD   = 'O'
	PARITY_EVEN  = 'E'
	PARITY_MARK  = 'M'
	PARITY_SPACE = 'S'
)

// 串口开放的接口
type Serial interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	InFlush() error
	OutFlush() error
	Close() error

	SetConfig(Config) error
	RestStart() error
}

type Config struct {
	Baud     uint32
	Size     byte
	Parity   byte
	StopBits byte

	MinByte  uint8
	ReadTime time.Duration
}
