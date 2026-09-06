//go:build linux

package tunnel

import (
	"fmt"
	"os"

	"golang.zx2c4.com/wireguard/tun"
)

type linuxDevice struct {
	device tun.Device
	name   string
	mtu    int
}

func NewDevice(name string, mtu int) (Device, error) {
	device, err := tun.CreateTUN(name, mtu)
	if err != nil {
		return nil, fmt.Errorf("create TUN device: %w", err)
	}

	actualName, err := device.Name()
	if err != nil {
		_ = device.Close()
		return nil, fmt.Errorf("get TUN device name: %w", err)
	}

	actualMTU, err := device.MTU()
	if err != nil {
		_ = device.Close()
		return nil, fmt.Errorf("get TUN device MTU: %w", err)
	}

	return &linuxDevice{
		device: device,
		name:   actualName,
		mtu:    actualMTU,
	}, nil
}

func (d *linuxDevice) Name() string {
	return d.name
}

func (d *linuxDevice) MTU() int {
	return d.mtu
}

func (d *linuxDevice) Read(buf []byte) (int, error) {
	bufs := [][]byte{buf}
	sizes := []int{0}

	_, err := d.device.Read(bufs, sizes, 0)
	if err != nil {
		return 0, err
	}

	return sizes[0], nil
}

func (d *linuxDevice) Write(buf []byte) (int, error) {
	return d.device.Write([][]byte{buf}, 0)
}

func (d *linuxDevice) Close() error {
	return d.device.Close()
}

func (d *linuxDevice) File() *os.File {
	return d.device.File()
}

var _ Device = (*linuxDevice)(nil)
