//go:build windows

package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

func listenSocket(network, address string) (net.Listener, error) {
	if network == "npipe" {
		return winio.ListenPipe(address, nil)
	}
	return net.Listen(network, address)
}

func dialSocket(network, address string) (net.Conn, error) {
	if network == "npipe" {
		return winio.DialPipe(address, nil)
	}
	return net.Dial(network, address)
}

func sendFD(conn net.Conn, f *os.File) error {
	return fmt.Errorf("FD passing not supported on windows")
}

func receiveFD(conn net.Conn) (*os.File, error) {
	return nil, fmt.Errorf("FD passing not supported on windows")
}
