package network

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

type MessageType byte

const (
	// MessageTypeStatus    MessageType = 0x1
	// MessageTypeGetStatus MessageType = 0x2
	MessageTypeGetCount MessageType = 0x1
)

type RPC struct {
	From    net.Addr
	Payload io.Reader
}
type Message struct {
	Header MessageType
	Data   []byte
}

func NewMessage(t MessageType, data []byte) *Message {
	return &Message{
		Header: t,
		Data:   data,
	}
}
func (msg *Message) Bytes() []byte {
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(msg)
	return buf.Bytes()
}

type DecodedMessage struct {
	From net.Addr
	Data any
}
type RPCDecodeFunc func(RPC) (*DecodedMessage, error)

func DefaultRPCDecodeFunc(rpc RPC) (*DecodedMessage, error) {
	msg := Message{}
	if err := gob.NewDecoder(rpc.Payload).Decode(&msg); err != nil {
		return nil, fmt.Errorf("failed to decode message from %s: %s", rpc.From, err)
	}

	// fmt.Printf("receiving message: %+v\n", msg.Header)

	switch msg.Header {
	case MessageTypeGetCount:
		var count GetCountMessage
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&count); err != nil {
			return nil, err
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: count,
		}, nil
	default:
		return nil, fmt.Errorf("invalid message header %x", msg.Header)
	}
}

type RPCProcessor interface {
	ProcessMessage(*DecodedMessage) error
}

func init() {
	gob.Register(elliptic.P256())
}
