package network

import (
	"io"
	"net"
)

type RPC struct {
	From    net.Addr
	Payload io.Reader
}
