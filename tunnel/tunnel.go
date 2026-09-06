package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/Aldiwildan77/tunny/route"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type Dialer interface {
	Dial(ctx context.Context, network, address string) (net.Conn, error)
}

type Tunnel struct {
	device Device
	dialer Dialer
	routes *route.Table

	stack    *stack.Stack
	endpoint *channel.Endpoint
}

func New(device Device, dialer Dialer, routes *route.Table) *Tunnel {
	return &Tunnel{
		device: device,
		dialer: dialer,
		routes: routes,
	}
}

func (t *Tunnel) Run(ctx context.Context) error {
	log.Printf(
		"tunnel started: device=%s mtu=%d",
		t.device.Name(),
		t.device.MTU(),
	)

	log.Printf("tunnel routes: %v", t.routes.GetIPs())

	cleanupRoutes, err := t.setupRoutes()
	if err != nil {
		return fmt.Errorf("setup routes: %w", err)
	}
	defer cleanupRoutes()

	if err := t.setupStack(ctx); err != nil {
		return fmt.Errorf("setup stack: %w", err)
	}

	defer t.close()

	go func() {
		if err := t.readDevice(ctx); err != nil && ctx.Err() == nil {
			log.Printf("TUN reader stopped: %v", err)
		}
	}()

	go func() {
		if err := t.writeDevice(ctx); err != nil && ctx.Err() == nil {
			log.Printf("TUN writer stopped: %v", err)
		}
	}()

	<-ctx.Done()

	return nil
}

func (t *Tunnel) setupStack(ctx context.Context) error {
	log.Println("setting up gVisor stack")

	t.endpoint = channel.New(
		1024,
		uint32(t.device.MTU()),
		"",
	)

	t.stack = stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
		},
	})

	if err := t.stack.CreateNIC(1, t.endpoint); err != nil {
		return fmt.Errorf("create NIC: %w", err)
	}

	log.Println("gVisor NIC created")

	// Accept packets regardless of whether the destination address
	// belongs to the userspace NIC.
	if err := t.stack.SetPromiscuousMode(1, true); err != nil {
		return fmt.Errorf("enable promiscuous mode: %w", err)
	}

	if err := t.stack.SetSpoofing(1, true); err != nil {
		return fmt.Errorf("enable spoofing: %w", err)
	}

	log.Println("gVisor promiscuous mode enabled")
	log.Println("gVisor spoofing enabled")

	subnet, err := tcpip.NewSubnet(
		tcpip.AddrFrom4([4]byte{0, 0, 0, 0}),
		tcpip.MaskFromBytes([]byte{0, 0, 0, 0}),
	)
	if err != nil {
		return fmt.Errorf("create default IPv4 route: %w", err)
	}

	t.stack.SetRouteTable([]tcpip.Route{
		{
			Destination: subnet,
			NIC:         1,
		},
	})

	log.Println("gVisor default IPv4 route installed")

	forwarder := tcp.NewForwarder(
		t.stack,
		0,
		64*1024,
		func(req *tcp.ForwarderRequest) {
			log.Println("TCP forwarder callback")

			go t.handleTCPForward(ctx, req)
		},
	)

	t.stack.SetTransportProtocolHandler(
		tcp.ProtocolNumber,
		forwarder.HandlePacket,
	)

	log.Println("TCP forwarder registered")

	return nil
}

func (t *Tunnel) readDevice(ctx context.Context) error {
	buf := make([]byte, t.device.MTU()+4)

	for {
		n, err := t.device.Read(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return fmt.Errorf("read TUN device: %w", err)
		}

		if n == 0 {
			continue
		}

		packet := append([]byte(nil), buf[:n]...)

		log.Printf(
			"TUN -> gVisor: packet=%d bytes",
			len(packet),
		)

		log.Printf(
			"TUN packet: len=%d first_byte=0x%02x",
			n,
			buf[0],
		)

		logIPv4Packet(packet)

		pkt := stack.NewPacketBuffer(
			stack.PacketBufferOptions{
				Payload: buffer.MakeWithData(packet),
			},
		)

		t.endpoint.InjectInbound(
			ipv4.ProtocolNumber,
			pkt,
		)

		pkt.DecRef()
	}
}

func (t *Tunnel) writeDevice(ctx context.Context) error {
	for {
		pkt := t.endpoint.ReadContext(ctx)
		if pkt == nil {
			return nil
		}

		slices := pkt.AsSlices()

		total := 0
		for _, data := range slices {
			total += len(data)
		}

		if total == 0 {
			pkt.DecRef()
			continue
		}

		packet := make([]byte, 0, total)

		for _, data := range slices {
			packet = append(packet, data...)
		}

		log.Printf(
			"gVisor -> TUN: packet=%d bytes first_byte=0x%02x",
			len(packet),
			packet[0],
		)

		_, err := t.device.Write(packet)
		if err != nil {
			log.Printf(
				"gVisor -> TUN write failed: bytes=%d first_byte=0x%02x err=%v",
				len(packet),
				packet[0],
				err,
			)

			// Drop this packet, but keep tunnel alive.
			pkt.DecRef()
			continue
		}

		pkt.DecRef()
	}
}

func (t *Tunnel) handleTCPForward(
	ctx context.Context,
	req *tcp.ForwarderRequest,
) {
	id := req.ID()

	log.Printf(
		"TCP request: src=%s:%d dst=%s:%d",
		id.RemoteAddress,
		id.RemotePort,
		id.LocalAddress,
		id.LocalPort,
	)

	defer req.Complete(true)

	var wq waiter.Queue

	log.Println("creating gVisor TCP endpoint")

	endpoint, tcpErr := req.CreateEndpoint(&wq)
	if tcpErr != nil {
		log.Printf(
			"create TCP endpoint failed: %v",
			tcpErr,
		)

		return
	}
	defer endpoint.Close()

	inbound := gonet.NewTCPConn(&wq, endpoint)
	defer inbound.Close()

	dstIP := id.LocalAddress.String()
	dst := net.JoinHostPort(
		id.LocalAddress.String(),
		fmt.Sprintf("%d", id.LocalPort),
	)

	domain := dstIP
	if host, ok := t.routes.GetHost(dstIP); ok {
		domain = host
	}

	log.Printf(
		"dialing outbound: %s",
		dst,
	)

	outbound, err := t.dialer.Dial(
		ctx,
		"tcp",
		dst,
	)
	if err != nil {
		log.Printf(
			"outbound dial failed: %s: %v",
			dst,
			err,
		)

		return
	}
	defer outbound.Close()

	log.Printf(
		"TCP request: %s:%d -> %s (%s)",
		id.RemoteAddress,
		id.RemotePort,
		domain,
		dst,
	)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		n, err := io.Copy(outbound, inbound)

		log.Printf(
			"client -> provider finished: bytes=%d err=%v",
			n,
			err,
		)
	}()

	go func() {
		defer wg.Done()

		n, err := io.Copy(inbound, outbound)

		log.Printf(
			"provider -> client finished: bytes=%d err=%v",
			n,
			err,
		)
	}()

	wg.Wait()

	log.Printf(
		"TCP forwarding finished: %s",
		dst,
	)
}

func (t *Tunnel) close() {
	log.Println("stopping tunnel")

	if t.stack != nil {
		t.stack.Close()
	}

	if t.endpoint != nil {
		t.endpoint.Close()
	}

	if err := t.device.Close(); err != nil {
		log.Printf(
			"close TUN device failed: %v",
			err,
		)
	}
}

func (t *Tunnel) setupRoutes() (func(), error) {
	router, ok := t.device.(RouteManager)
	if !ok {
		return func() {}, fmt.Errorf(
			"device %s does not support routing",
			t.device.Name(),
		)
	}

	var added []net.IP

	for _, ip := range t.routes.GetNetIPs() {
		ip = ip.To4()
		if ip == nil {
			continue
		}

		log.Printf(
			"adding OS route: %s -> %s",
			ip,
			t.device.Name(),
		)

		if err := router.AddRoute(ip); err != nil {
			for _, addedIP := range added {
				if removeErr := router.RemoveRoute(addedIP); removeErr != nil {
					log.Printf(
						"cleanup route %s failed: %v",
						addedIP,
						removeErr,
					)
				}
			}

			return func() {}, err
		}

		added = append(added, ip)

		log.Printf(
			"OS route added: %s -> %s",
			ip,
			t.device.Name(),
		)
	}

	cleanup := func() {
		for _, ip := range added {
			log.Printf(
				"removing OS route: %s",
				ip,
			)

			if err := router.RemoveRoute(ip); err != nil {
				log.Printf(
					"remove route %s failed: %v",
					ip,
					err,
				)
			}
		}
	}

	return cleanup, nil
}

func logIPv4Packet(packet []byte) {
	if len(packet) < 20 {
		log.Printf("packet too small: %d bytes", len(packet))
		return
	}

	version := packet[0] >> 4
	ihl := int(packet[0]&0x0f) * 4

	if version != 4 || len(packet) < ihl {
		log.Printf("not valid IPv4 packet: version=%d len=%d", version, len(packet))
		return
	}

	protocol := packet[9]

	src := net.IP(packet[12:16])
	dst := net.IP(packet[16:20])

	log.Printf(
		"IPv4 packet: %s -> %s protocol=%d len=%d",
		src,
		dst,
		protocol,
		len(packet),
	)
}
