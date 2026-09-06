//go:build darwin

package tunnel

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

const (
	darwinPacketOffset = 4

	tunnelLocalIP  = "169.254.100.1"
	tunnelRemoteIP = "169.254.100.2"
	tunnelNetmask  = "255.255.255.252"
)

type darwinDevice struct {
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

	d := &darwinDevice{
		device: device,
		name:   actualName,
		mtu:    actualMTU,
	}

	if err := d.configure(); err != nil {
		_ = d.Close()
		return nil, err
	}

	return d, nil
}

func (d *darwinDevice) configure() error {
	cmd := exec.Command(
		"ifconfig",
		d.name,
		tunnelLocalIP,
		tunnelRemoteIP,
		"netmask",
		tunnelNetmask,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"configure TUN device %s: %w: %s",
			d.name,
			err,
			output,
		)
	}

	return nil
}

func (d *darwinDevice) AddRoute(ip net.IP) error {
	ip = ip.To4()
	if ip == nil {
		return fmt.Errorf("only IPv4 routes are supported: %v", ip)
	}

	cmd := exec.Command(
		"route",
		"-n",
		"add",
		"-host",
		ip.String(),
		"-interface",
		d.name,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"add route %s via %s: %w: %s",
			ip,
			d.name,
			err,
			output,
		)
	}

	return nil
}

func (d *darwinDevice) RemoveRoute(ip net.IP) error {
	ip = ip.To4()
	if ip == nil {
		return nil
	}

	cmd := exec.Command(
		"route",
		"-n",
		"delete",
		"-host",
		ip.String(),
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"remove route %s: %w: %s",
			ip,
			err,
			output,
		)
	}

	return nil
}

func (d *darwinDevice) Name() string {
	return d.name
}

func (d *darwinDevice) MTU() int {
	return d.mtu
}

func (d *darwinDevice) Read(buf []byte) (int, error) {
	if len(buf) < d.mtu+darwinPacketOffset {
		return 0, fmt.Errorf(
			"buffer too small: got %d, need at least %d",
			len(buf),
			d.mtu+darwinPacketOffset,
		)
	}

	// wireguard-go expects the caller to reserve 4 bytes before the
	// packet for the utun address-family header.
	//
	// We don't want that header exposed to the tunnel layer, so read
	// into a temporary buffer and copy only the IP packet to buf[0:].
	raw := buf[:d.mtu+darwinPacketOffset]

	bufs := [][]byte{raw}
	sizes := []int{0}

	_, err := d.device.Read(
		bufs,
		sizes,
		darwinPacketOffset,
	)
	if err != nil {
		return 0, err
	}

	n := sizes[0]

	if n <= 0 {
		return 0, nil
	}

	copy(buf, raw[darwinPacketOffset:darwinPacketOffset+n])

	return n, nil
}

func (d *darwinDevice) Write(buf []byte) (int, error) {
	packet := make([]byte, len(buf)+4)

	copy(packet[4:], buf)

	_, err := d.device.Write(
		[][]byte{packet},
		4,
	)
	if err != nil {
		return 0, err
	}

	return len(buf), nil
}

func (d *darwinDevice) Close() error {
	return d.device.Close()
}

func (d *darwinDevice) File() *os.File {
	return d.device.File()
}

var _ Device = (*darwinDevice)(nil)
var _ RouteManager = (*darwinDevice)(nil)
