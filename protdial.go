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

		port := binary.BigEndian.Uint16(portBytes[1:])

		portsMap.mux.Lock()
		portContext := portsMap.ports[port]
		portsMap.mux.Unlock()

		if portContext == nil {
			s.Reset()
			return
		}

		var conn net.Conn

		switch protocolType {
		case protocolTypeTCP:
			conn, err = net.DialTCP("tcp", nil, &net.TCPAddr{
				IP:   nil,
				Port: int(port),
			})
		case protocolTypeUDP:
			conn, err = net.DialUDP("udp", nil, &net.UDPAddr{
				IP:   nil,
				Port: int(port),
			})
		}

		if err != nil {
			s.Reset()
			onErrFn(fmt.Errorf("dial handler: %s", err))
			return
		}

		pipeBothIOs(portContext, s, conn)
	})
}

func (f *Forwarder) dial(ctx context.Context, peerid peer.ID, protocolType byte, listenip string, port uint16) {
	switch protocolType {
	case protocolTypeTCP:
		lport := int(port)
		ln, err := net.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.ParseIP(listenip),
			Port: lport,
		})
		if err != nil {
			onErrFn(fmt.Errorf("dial tcp: %s", err))

			for i := 0; i < 4; i++ {
				lport = rand.Intn(65535-1024) + 1024

				ln, err = net.ListenTCP("tcp", &net.TCPAddr{
					IP:   net.ParseIP(listenip),
					Port: lport,
				})

				if err != nil {
					onErrFn(fmt.Errorf("dial tcp: %s", err))
				} else {
					break
				}
			}

			if err != nil {
				return
			}
		}

		addressstr := "tcp:" + listenip + ":" + strconv.Itoa(lport) + " -> " + strconv.FormatUint(uint64(port), 10)

		onInfoFn("Listening " + addressstr)

		go func() {
		loop:
			for {
				conn, err := ln.Accept()
				if err != nil {
					onErrFn(fmt.Errorf("dial tcp ln accept: %s", err))
					select {
					case <-ctx.Done():
						break loop
					default:
						continue loop
					}
				}

				go f.handleDialStream(ctx, peerid, conn, protocolTypeTCP, port)
			}
		}()

		<-ctx.Done()
		ln.Close()

		onInfoFn("Closed " + addressstr)

	case protocolTypeUDP:
		lport := int(port)

		conn, err := net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.ParseIP(listenip),
			Port: lport,
		})

		if err != nil {
			onErrFn(fmt.Errorf("dial udp ln: %s", err))

			for i := 0; i < 4; i++ {
				lport = rand.Intn(65535-1024) + 1024

				conn, err = net.ListenUDP("udp", &net.UDPAddr{
					IP:   net.ParseIP(listenip),
					Port: lport,
				})

				if err != nil {
					onErrFn(fmt.Errorf("dial udp: %s", err))
				} else {
					break
				}
			}

			if err != nil {
				return
			}
		}

		addressstr := "udp:" + listenip + ":" + strconv.Itoa(lport) + " -> " + strconv.FormatUint(uint64(port), 10)

		onInfoFn("Listening " + addressstr)

		go func() {
		loop:
			for {
				select {
				case <-ctx.Done():
					break loop
				default:
					f.handleDialStream(ctx, peerid, conn, protocolTypeUDP, port)
				}
			}
		}()

		<-ctx.Done()
		conn.Close()

		onInfoFn("Closed " + addressstr)
	}
}

func (f *Forwarder) handleDialStream(ctx context.Context, peerid peer.ID, conn net.Conn, protocolType byte, port uint16) {
	s, err := f.host.NewStream(ctx, peerid, dialProtID)
	if err != nil {
		conn.Close()
		onErrFn(fmt.Errorf("startDialStream: %s", err))
		return
	}

	a := make([]byte, 1)

	a[0] = protocolType

	_, err = s.Write(a)
	if err != nil {
		s.Close()
		conn.Close()
		onErrFn(fmt.Errorf("startDialStream: %s", err))
		return
	}

	b := make([]byte, 2)

	binary.BigEndian.PutUint16(b, port)

	_, err = s.Write(b)
	if err != nil {
		s.Close()
		conn.Close()
		onErrFn(fmt.Errorf("startDialStream: %s", err))
		return
	}

	pipeBothIOs(ctx, conn, s)
}

func pipeBothIOs(parentCtx context.Context, a io.ReadWriteCloser, b io.ReadWriteCloser) {
	ctx, cancel := context.WithCancel(parentCtx)

	go func() {
		_, err := io.Copy(b, a)
		b.Close()
		if err != nil {
			cancel()
			onErrFn(fmt.Errorf("pipeBothIOs b<-a: %s", err))
		}
	}()
	go func() {
		_, err := io.Copy(a, b)
		a.Close()
		if err != nil {
			cancel()
			onErrFn(fmt.Errorf("pipeBothIOs a<-b: %s", err))
		}
	}()

	select {
	case <-parentCtx.Done():
		a.Close()
		b.Close()
	case <-ctx.Done():
	}
}
