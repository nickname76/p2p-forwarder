package p2pforwarder

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	// ErrMaxConnections = error "Max connections reached"
	ErrMaxConnections = errors.New("Max connections reached")
	// ErrPortAlreadyOpened = error "Port already opened"
	ErrPortAlreadyOpened = errors.New("Port already opened")
	// ErrUnknownNetworkType = error "Unkown network type. It must be \"tcp\" or \"udp\""
	ErrUnknownNetworkType = errors.New("Unkown network type. It must be \"tcp\" or \"udp\"")
)

// OpenPort opens port in specified networkType - "tcp" or "udp"
func (f *Forwarder) OpenPort(networkType string, port uint16) (cancel func(), err error) {
	switch networkType {
	case "tcp":
		cancel, err = f.addOpenPort(f.openPorts.tcp, port)
	case "udp":
		cancel, err = f.addOpenPort(f.openPorts.udp, port)
	default:
		cancel, err = nil, ErrUnknownNetworkType
		return
	}

	if err == nil {
		go f.publishOpenPortsManifest()
	}

	return cancel, err
}

func (f *Forwarder) addOpenPort(portsMap *openPortsStoreMap, port uint16) (cancel func(), err error) {
	portsMap.mux.Lock()

	if portsMap.ports[port] != nil {
		portsMap.mux.Unlock()
		return nil, ErrPortAlreadyOpened
	}

	var cancelfn func()
	portsMap.ports[port], cancelfn = context.WithCancel(context.Background())

	portsMap.mux.Unlock()

	cancel = func() {
		portsMap.mux.Lock()
		cancelfn()
		delete(portsMap.ports, port)
		portsMap.mux.Unlock()

		go f.publishOpenPortsManifest()
	}

	return cancel, nil
}

var (
	listenipN    int
	listenipNMux sync.Mutex
)

// Forward starts forwarding connections to `listenip`:`PORT` to passed id`:`PORT`
func (f *Forwarder) Forward(id string) (listenip string, cancel context.CancelFunc, err error) {
	peerid, err := peer.IDB58Decode(id)
	if err != nil {
		return "", nil, err
	}

	listenipNMux.Lock()

	if listenipN > 255 {
		return "", nil, ErrMaxConnections
	}
	listenipN++

	listenip = "127.0.89." + strconv.Itoa(listenipN)

	listenipNMux.Unlock()

	f.portsSubscriptionsMux.Lock()

	subCh := make(chan *portsManifest, 5)
	f.portsSubscriptions[peerid] = subCh

	f.portsSubscriptionsMux.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		var (
			tcpPortsOld = make(map[uint16]func())
			udpPortsOld = make(map[uint16]func())
		)

	loop:
		for {
			select {
			case <-ctx.Done():
				f.portsSubscriptionsMux.Lock()
				close(subCh)
				delete(f.portsSubscriptions, peerid)
				f.portsSubscriptionsMux.Unlock()
				break loop
			case portsM := <-subCh:
				if portsM.tcp != nil {
					f.updatePortsListening(ctx, protocolTypeTCP, portsM.tcp, &tcpPortsOld, peerid, listenip)
				}

				if portsM.udp != nil {
					f.updatePortsListening(ctx, protocolTypeUDP, portsM.udp, &udpPortsOld, peerid, listenip)
				}
			}
		}
	}()

	s, err := f.host.NewStream(ctx, peerid, portssubProtID)
	if err != nil {
		cancel()
		return "", nil, err
	}
	// This starts subscription
	_, err = s.Write([]byte{portssubModeSubscribe})
	if err != nil {
		cancel()
		return "", nil, err
	}

	s.Close()

	return listenip, cancel, nil
}

func (f *Forwarder) updatePortsListening(parentCtx context.Context, protocolType byte, portsArr []uint16, portsOld *map[uint16]func(), peerid peer.ID, listenip string) {
	ports := make(map[uint16]func())

	for _, port := range portsArr {
		cancel, ok := (*portsOld)[port]

		if ok {
			ports[port] = cancel
			delete(*portsOld, port)
			continue
		}

		var ctx context.Context
		ctx, ports[port] = context.WithCancel(parentCtx)

		go f.dial(ctx, peerid, protocolType, listenip, port)
	}

	for _, v := range *portsOld {
		v()
	}

	*portsOld = ports
}
