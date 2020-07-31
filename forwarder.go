package p2pforwarder

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	autonat "github.com/libp2p/go-libp2p-autonat"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	routing "github.com/libp2p/go-libp2p-routing"
	secio "github.com/libp2p/go-libp2p-secio"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/sparkymat/appdir"
)

const (
	protocolTypeTCP byte = 0x00
	protocolTypeUDP byte = 0x01
)

// Forwarder - instance of P2P Forwarder
type Forwarder struct {
	host      host.Host
	openPorts *openPortsStore

	portsSubscriptions    map[peer.ID]chan *portsManifest
	portsSubscriptionsMux sync.Mutex

	portsSubscribers    map[peer.ID]struct{}
	portsSubscribersMux sync.Mutex
}

type openPortsStore struct {
	tcp *openPortsStoreMap
	udp *openPortsStoreMap
}

type openPortsStoreMap struct {
	ports map[uint16]context.Context
	mux   sync.Mutex
}

func newOpenPortsStore() *openPortsStore {
	return &openPortsStore{
		tcp: &openPortsStoreMap{
			ports: map[uint16]context.Context{},
		},
		udp: &openPortsStoreMap{
			ports: map[uint16]context.Context{},
		},
	}
}

// NewForwarder - instances Forwarder and connects it to libp2p network
func NewForwarder() (*Forwarder, context.CancelFunc, error) {
	priv, err := loadUserPrivKey()
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	h, err := createLibp2pHost(ctx, priv)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	anatPriv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	hautonat, err := createLibp2pHost(ctx, anatPriv)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	_, err = autonat.New(ctx, h, autonat.EnableService(hautonat.Network()))
	if err != nil {
		cancel()
		return nil, nil, err
	}

	// The last step to get fully up and running would be to connect to
	// bootstrap peers (or any other peers).

	// This connects to public bootstrappers
	for _, addr := range dht.DefaultBootstrapPeers {
		pi, _ := peer.AddrInfoFromP2pAddr(addr)
		// We ignore errors as some bootstrap peers may be down
		// and that is fine.
		h.Connect(ctx, *pi)
	}

	f := &Forwarder{
		host: h,

		openPorts:          newOpenPortsStore(),
		portsSubscriptions: make(map[peer.ID]chan *portsManifest),
		portsSubscribers:   make(map[peer.ID]struct{}),
	}

	setDialHandler(f)
	setPortsPubSubHandler(f)

	return f, cancel, nil
}

func loadUserPrivKey() (priv crypto.PrivKey, err error) {
	krPath, err := appdir.AppInfo{
		Author: "nickname32",
		Name:   "P2P Forwarder",
	}.ConfigPath("keypair")
	if err != nil {
		return nil, err
	}

	pkFile, err := os.Open(krPath)

	if err == nil {
		defer pkFile.Close()

		b, err := ioutil.ReadAll(pkFile)
		if err != nil {
			return nil, err
		}

		priv, err = crypto.UnmarshalPrivateKey(b)
		if err != nil {
			return nil, err
		}

		return priv, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	priv, _, err = crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, err
	}
	b, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Dir(krPath), os.ModePerm)
	if err != nil {
		return nil, err
	}
	newPkFile, err := os.Create(krPath)
	if err != nil {
		return nil, err
	}
	_, err = newPkFile.Write(b)
	if err != nil {
		return nil, err
	}
	err = newPkFile.Close()
	if err != nil {
		return nil, err
	}

	return priv, nil
}

func createLibp2pHost(ctx context.Context, priv crypto.PrivKey) (host.Host, error) {
	return libp2p.New(ctx,
		// Use the keypair
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9000",      // regular tcp connections
			"/ip4/0.0.0.0/udp/9000/quic", // a UDP endpoint for the QUIC transport
		),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support secio connections
		libp2p.Security(secio.ID, secio.New),
		// support QUIC - experimental
		libp2p.Transport(libp2pquic.NewTransport),
		// support any other default transports (TCP)
		libp2p.DefaultTransports,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connmgr.NewConnManager(
			100,         // Lowwater
			400,         // HighWater,
			time.Minute, // GracePeriod
		)),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			return dht.New(ctx, h)
		}),
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		libp2p.EnableAutoRelay(),
	)
}

// ID returns id of Forwarder
func (f *Forwarder) ID() string {
	return f.host.ID().String()
}

var onErrFn = func(err error) {
	println(err.Error())
}
var onInfoFn = func(str string) {
	println(str)
}

// OnError sets function which be called on error inside this package
func OnError(fn func(error)) {
	if fn == nil {
		return
	}
	onErrFn = fn
}

// OnInfo sets function which be called on information inside this package
func OnInfo(fn func(string)) {
	if fn == nil {
		return
	}
	onInfoFn = fn
}
