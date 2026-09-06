package tunnel

import (
	"testing"
)

func TestCreateDevice(t *testing.T) {
	device, err := NewDevice("utun", 1500)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	t.Logf("device=%s mtu=%d", device.Name(), device.MTU())
}

func TestDeviceProperties(t *testing.T) {
	device, err := NewDevice("utun", 1500)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	if device.Name() == "" {
		t.Fatal("device name is empty")
	}

	if device.MTU() <= 0 {
		t.Fatalf("invalid MTU: %d", device.MTU())
	}
}
