package p2p

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/czh0526/perception/rlp"
	"github.com/libp2p/go-libp2p-core/network"
)

type ProtoRW struct {
	stream   network.Stream
	rw       *bufio.ReadWriter
	rmu, wmu sync.Mutex
}

func NewProtoRW(stream network.Stream) *ProtoRW {
	return &ProtoRW{
		stream: stream,
		rw:     bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)),
	}
}

func (c *ProtoRW) Close(err error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
		SendItems(c, discMsg, r)
		fmt.Println(`\/`)
		fmt.Println(`/\`)
	}

	c.stream.Close()
}

// 从 stream 中读取并反序列化 Msg
func (c *ProtoRW) ReadMsg() (Msg, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	msg := Msg{}
	if err := rlp.Decode(c.rw, &msg.Code); err != nil {
		return msg, err
	}

	log.Printf("\t\t %v <== [%v] \n", msgType(msg.Code), c.stream.Protocol())

	if err := rlp.Decode(c.rw, &msg.Size); err != nil {
		return msg, err
	}

	var payload = make([]byte, msg.Size)
	if _, err := io.ReadFull(c.rw, payload); err != nil {
		return msg, err
	}

	msg.Payload = bytes.NewReader(payload)
	return msg, nil
}

// 将 Msg 序列化并写入 stream
func (c *ProtoRW) WriteMsg(msg Msg) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	log.Printf("\t\t %v ==> [%v] \n", msgType(msg.Code), c.stream.Protocol())

	num := 0
	ptype, err := rlp.EncodeToBytes(msg.Code)
	if err != nil {
		return err
	}
	written, err := c.rw.Write(ptype)
	if err != nil {
		return err
	}
	num += written

	psize, err := rlp.EncodeToBytes(msg.Size)
	if err != nil {
		return err
	}
	written, err = c.rw.Write(psize)
	if err != nil {
		return err
	}
	num += written

	copy, err := io.Copy(c.rw, msg.Payload)
	if err != nil {
		return err
	}
	num += int(copy)
	c.rw.Flush()

	return nil
}
