package p2p

import (
	"fmt"
	"log"

	"github.com/czh0526/perception/rlp"
)

func perceptionHandshake(rw *ProtoRW, our *protoHandshake) (their *protoHandshake, err error) {
	log.Println("1). do 'perception' handshake ...")
	werr := make(chan error, 1)
	go func() {
		werr <- Send(rw, handshakeMsg, our)
	}()
	if their, err = readHandshakeMsg(rw); err != nil {
		<-werr
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	return their, nil
}

func readHandshakeMsg(rw MsgReader) (*protoHandshake, error) {
	msg, err := rw.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.Size > baseProtocolMaxMsgSize {
		return nil, fmt.Errorf("message to big")
	}
	if msg.Code == discMsg {
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return nil, reason[0]
	}
	if msg.Code != handshakeMsg {
		return nil, fmt.Errorf("expected handshake, got %x", msg.Code)
	}

	var hs protoHandshake
	if err := msg.Decode(&hs); err != nil {
		return nil, err
	}
	if len(hs.PubKey) != 33 {
		return nil, DiscInvalidIdentity
	}
	return &hs, nil
}
