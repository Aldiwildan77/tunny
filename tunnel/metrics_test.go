package tunnel

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Aldiwildan77/tunny/observability"
	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
)

func TestMain(m *testing.M) {
	observabilitytest.Install()
	os.Exit(m.Run())
}

func TestPacketAndForwardMetrics(t *testing.T) {
	ctx := context.Background()
	recordPacket(ctx, observability.DirectionReceive, 60, nil)
	recordPacket(ctx, observability.DirectionTransmit, 40, nil)
	recordPacket(ctx, observability.DirectionTransmit, 40, errors.New("write failed"))
	recordForwardError(ctx, "udp", errDial)

	assert.Equal(t, map[string]float64{
		"network.io.direction=receive,tunny.outcome=success":                        1,
		"network.io.direction=transmit,tunny.outcome=success":                       1,
		"error.type=device_write,network.io.direction=transmit,tunny.outcome=error": 1,
	}, observabilitytest.Points(t, "tunny.tunnel.packets"))
	assert.Equal(t, map[string]float64{
		"network.io.direction=receive":  60,
		"network.io.direction=transmit": 40,
	}, observabilitytest.Points(t, "tunny.tunnel.io"))
	assert.Equal(t, map[string]float64{
		"error.type=dial,network.transport=udp": 1,
	}, observabilitytest.Points(t, "tunny.tunnel.forward.errors"))
}
