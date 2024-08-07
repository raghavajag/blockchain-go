package network

import (
	"blockchain/blockchain"
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

type MessageType byte

const (
	MessageTypeTx        MessageType = 0x1
	MessageTypeBlock     MessageType = 0x2 // broadcasting the mined block
	MessageTypeGetBlocks MessageType = 0x3
	MessageTypeStatus    MessageType = 0x4
	MessageTypeGetStatus MessageType = 0x5
	MessageTypeBlocks    MessageType = 0x6 // sending blocks to other nodes
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
	case MessageTypeGetStatus:
		return &DecodedMessage{
			From: rpc.From,
			Data: &GetStatusMessage{},
		}, nil
	case MessageTypeStatus:
		statusMessage := new(StatusMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(statusMessage); err != nil {
			return nil, fmt.Errorf("failed to decode status message: %s", err)
		}
		return &DecodedMessage{
			From: rpc.From,
			Data: statusMessage,
		}, nil
	case MessageTypeGetBlocks:
		getBlocks := new(GetBlocksMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(getBlocks); err != nil {
			return nil, err
		}

		return &DecodedMessage{
			From: rpc.From,
			Data: getBlocks,
		}, nil
	case MessageTypeBlocks:
		blocks := new(BlocksMessage)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(blocks); err != nil {
			return nil, err
		}

		return &DecodedMessage{
			From: rpc.From,
			Data: blocks,
		}, nil
	case MessageTypeBlock:
		block := new(blockchain.Block)
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(block); err != nil {
			return nil, err
		}

		return &DecodedMessage{
			From: rpc.From,
			Data: block,
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
