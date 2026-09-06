package tunnel

import (
	"io"
	"net"
	"os"
)

type Device interface {
	io.ReadWriteCloser

	Name() string
	MTU() int
	File() *os.File
}

type RouteManager interface {
	AddRoute(ip net.IP) error
	RemoveRoute(ip net.IP) error
}
