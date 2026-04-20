//go:build !windows

package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func listenSocket(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func dialSocket(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func sendFD(conn net.Conn, f *os.File) error {
	uconn, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("not a unix connection")
	}
	rights := syscall.UnixRights(int(f.Fd()))
	_, _, err := uconn.WriteMsgUnix([]byte{0}, rights, nil)
	return err
}

func receiveFD(conn net.Conn) (*os.File, error) {
	uconn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("not a unix connection")
	}
	buf := make([]byte, 1)
	oob := make([]byte, 32)
	n, oobn, _, _, err := uconn.ReadMsgUnix(buf, oob)
	if err != nil || n == 0 {
		return nil, err
	}
	msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil || len(msgs) == 0 {
		return nil, err
	}
	fds, err := syscall.ParseUnixRights(&msgs[0])
	if err != nil || len(fds) == 0 {
		return nil, err
	}
	return os.NewFile(uintptr(fds[0]), "proxied-socket"), nil
}
