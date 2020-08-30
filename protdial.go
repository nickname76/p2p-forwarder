package p2pforwarder

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pion/udp"
)

const dialProtID protocol.ID = "/p2pforwarder/dial/1.0.0"

func setDialHandler(f *Forwarder) {
	f.host.SetStreamHandler(dialProtID, func(s network.Stream) {
		onInfoFn("'dial' from " + s.Conn().RemotePeer().String())

		portBytes := make([]byte, 3)
		_, err := io.ReadFull(s, portBytes)
		if err != nil {
			s.Reset()
			onErrFn(fmt.Errorf("dial handler: %s", err))
			return
		}

		protocolType := portBytes[0]
		port := binary.BigEndian.Uint16(portBytes[1:])

		var portsMap *openPortsStoreMap
		switch protocolType {
		case protocolTypeTCP:
			portsMap = f.openPorts.tcp
		case protocolTypeUDP:
			portsMap = f.openPorts.udp
		default:
			s.Reset()
			return
		}

		portsMap.mux.Lock()
		portContext := portsMap.ports[port]
		portsMap.mux.Unlock()

		if portContext == nil {
			s.Reset()
			return
		}

		var conn net.Conn

		var addr string
		portInt := int(port)
		switch protocolType {
		case protocolTypeTCP:
			addr = "tcp:" + strconv.Itoa(portInt)

			conn, err = net.DialTCP("tcp", nil, &net.TCPAddr{
				IP:   nil,
				Port: portInt,
			})
		case protocolTypeUDP:
			addr = "udp:" + strconv.Itoa(portInt)

			conn, err = net.DialUDP("udp", nil, &net.UDPAddr{
				IP:   nil,
				Port: portInt,
			})
		}
		onInfoFn("Dialing to " + addr + " from " + s.Conn().RemotePeer().String())
		if err != nil {
			s.Reset()
			onErrFn(fmt.Errorf("dial handler: %s", err))
			return
		}

		pipeBothIOs(portContext, s, conn)

		s.Close()
		conn.Close()
	})
}

func createAddrInfoString(network string, listenip string, lport int, port int) string {
	return network + ":" + listenip + ":" + strconv.Itoa(lport) + " -> " + network + ":" + strconv.Itoa(port)
}

func (f *Forwarder) dial(ctx context.Context, peerid peer.ID, protocolType byte, listenip string, port uint16) {
	lport := int(port)

	var addressinfostr string

	var lnfunc func(lip net.IP, port int) (net.Listener, error)

	switch protocolType {
	case protocolTypeTCP:
		addressinfostr = createAddrInfoString("tcp", listenip, lport, int(port))

		lnfunc = func(lip net.IP, port int) (net.Listener, error) {
			return net.ListenTCP("tcp", &net.TCPAddr{
				IP:   lip,
				Port: port,
			})
		}
	case protocolTypeUDP:
		addressinfostr = createAddrInfoString("udp", listenip, lport, int(port))

		lnfunc = func(lip net.IP, port int) (net.Listener, error) {
			return udp.Listen("udp", &net.UDPAddr{
				IP:   lip,
				Port: port,
			})
		}
	}

	lip := net.ParseIP(listenip)

	ln, err := lnfunc(lip, lport)
	if err != nil {
		onErrFn(fmt.Errorf("dial: %s", err))

		for i := 0; i < 4; i++ {
			lport = rand.Intn(65535-1024) + 1024

			ln, err = lnfunc(lip, lport)

			if err != nil {
				onErrFn(fmt.Errorf("dial: %s", err))
			} else {
				break
			}
		}

		if err != nil {
			return
		}
	}

	onInfoFn("Listening " + addressinfostr)

	go func() {
	loop:
		for {
			conn, err := ln.Accept()
			if err != nil {
				onErrFn(fmt.Errorf("dial: %s", err))
				select {
				case <-ctx.Done():
					break loop
				default:
					continue loop
				}
			}

			go func() {
				defer conn.Close()

				s, err := f.host.NewStream(ctx, peerid, dialProtID)
				if err != nil {
					onErrFn(fmt.Errorf("dial: %s", err))
					return
				}
				defer s.Close()

				p := make([]byte, 3)
				p[0] = protocolType
				binary.BigEndian.PutUint16(p[1:3], port)

				_, err = s.Write(p)
				if err != nil {
					onErrFn(fmt.Errorf("dial: %s", err))
					return
				}

				pipeBothIOs(ctx, conn, s)
			}()
		}
	}()

	<-ctx.Done()
	ln.Close()

	onInfoFn("Closed " + addressinfostr)
}

func pipeBothIOs(ctx context.Context, a io.ReadWriter, b io.ReadWriter) {
	copyctx, cancel := context.WithCancel(ctx)

	go func() {
		_, err := io.Copy(b, a)
		cancel()
		if err != nil {
			onErrFn(fmt.Errorf("pipeBothIOs b<-a: %s", err))
		}
	}()
	go func() {
		_, err := io.Copy(a, b)
		cancel()
		if err != nil {
			onErrFn(fmt.Errorf("pipeBothIOs a<-b: %s", err))
		}
	}()

	<-copyctx.Done()
}
