package p2p

import (
	"fmt"
	"io"
	"time"

	"github.com/czh0526/perception/rlp"
)

type Msg struct {
	Code       uint64
	Size       uint32
	Payload    io.Reader
	ReceivedAt time.Time
}

func (msg Msg) String() string {
	return fmt.Sprintf("%v #%v (%v bytes)", msgType(msg.Code), msg.Code, msg.Size)
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	WriteMsg(Msg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewStream(msg.Payload, uint64(msg.Size))
	if err := s.Decode(val); err != nil {
		return fmt.Errorf("invalid Msg, err = %v", err)
	}
	return nil
}

func Send(w MsgWriter, msgcode uint64, data interface{}) error {
	size, r, err := rlp.EncodeToReader(data)
	if err != nil {
		return err
	}
	return w.WriteMsg(Msg{Code: msgcode, Size: uint32(size), Payload: r})
}

func SendItems(w MsgWriter, msgcode uint64, elems ...interface{}) error {
	return Send(w, msgcode, elems)
}
