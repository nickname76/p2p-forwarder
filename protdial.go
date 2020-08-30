package p2pforwarder

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"

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

func (f *Forwarder) dial(ctx context.Context, peerid peer.ID, protocolType byte, listenip string, port uint16) {
	switch protocolType {
	case protocolTypeTCP:
		f.dialTCP(ctx, peerid, protocolType, listenip, port)
	case protocolTypeUDP:
		f.dialUDP(ctx, peerid, protocolType, listenip, port)
	}
}

func (f *Forwarder) dialTCP(ctx context.Context, peerid peer.ID, protocolType byte, listenip string, port uint16) {
	lport := int(port)
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP(listenip),
		Port: lport,
	})
	if err != nil {
		onErrFn(fmt.Errorf("dialTCP: %s", err))

		for i := 0; i < 4; i++ {
			lport = rand.Intn(65535-1024) + 1024

			ln, err = net.ListenTCP("tcp", &net.TCPAddr{
				IP:   net.ParseIP(listenip),
				Port: lport,
			})

			if err != nil {
				onErrFn(fmt.Errorf("dialTCP: %s", err))
			} else {
				break
			}
		}

		if err != nil {
			return
		}
	}

	addressstr := "tcp:" + listenip + ":" + strconv.Itoa(lport) + " -> " + "tcp:" + strconv.FormatUint(uint64(port), 10)

	onInfoFn("Listening " + addressstr)

	go func() {
	loop:
		for {
			conn, err := ln.Accept()
			if err != nil {
				onErrFn(fmt.Errorf("dialTCP: %s", err))
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
					onErrFn(fmt.Errorf("dialTCP: %s", err))
					return
				}
				defer s.Close()

				p := make([]byte, 3)
				p[0] = protocolType
				binary.BigEndian.PutUint16(p[1:3], port)

				_, err = s.Write(p)
				if err != nil {
					onErrFn(fmt.Errorf("dialTCP: %s", err))
					return
				}

				pipeBothIOs(ctx, conn, s)
			}()
		}
	}()

	<-ctx.Done()
	ln.Close()

	onInfoFn("Closed " + addressstr)
}

type udpConnAddrWriter struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func (ucaw *udpConnAddrWriter) Write(p []byte) (int, error) {
	return ucaw.conn.WriteToUDP(p, ucaw.addr)
}

func (f *Forwarder) dialUDP(ctx context.Context, peerid peer.ID, protocolType byte, listenip string, port uint16) {
	lport := int(port)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(listenip),
		Port: lport,
	})

	if err != nil {
		onErrFn(fmt.Errorf("dialUDP: %s", err))

		for i := 0; i < 4; i++ {
			lport = rand.Intn(65535-1024) + 1024

			conn, err = net.ListenUDP("udp", &net.UDPAddr{
				IP:   net.ParseIP(listenip),
				Port: lport,
			})

			if err != nil {
				onErrFn(fmt.Errorf("dialUDP: %s", err))
			} else {
				break
			}
		}

		if err != nil {
			return
		}
	}

	addressstr := "udp:" + listenip + ":" + strconv.Itoa(lport) + " -> " + "udp:" + strconv.FormatUint(uint64(port), 10)

	onInfoFn("Listening " + addressstr)

	var (
		buf      = make([]byte, 1024)
		conns    = map[string]network.Stream{}
		connsMux sync.Mutex
	)
	go func() {
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			default:
				n, udpaddr, err := conn.ReadFromUDP(buf)
				if err != nil {
					onErrFn(fmt.Errorf("dialUDP: %s", err))
					continue loop
				}

				connsMux.Lock()
				s, ok := conns[udpaddr.String()]
				if !ok {
					s, err = f.host.NewStream(ctx, peerid, dialProtID)
					if err != nil {
						connsMux.Unlock()
						onErrFn(fmt.Errorf("dialUDP: %s", err))
						continue loop
					}

					p := make([]byte, 3)
					p[0] = protocolType
					binary.BigEndian.PutUint16(p[1:3], port)

					_, err = s.Write(p)
					if err != nil {
						connsMux.Unlock()
						s.Close()
						onErrFn(fmt.Errorf("dialUDP: %s", err))
						continue loop
					}

					conns[udpaddr.String()] = s

					go func() {
						_, err := io.Copy(&udpConnAddrWriter{
							conn: conn,
							addr: udpaddr,
						}, s)
						if err != nil {
							onErrFn(fmt.Errorf("dialUDP: %s", err))
						}

						s.Close()

						connsMux.Lock()
						delete(conns, udpaddr.String())
						connsMux.Unlock()

					}()
				}
				connsMux.Unlock()

				_, err = s.Write(buf[:n])
				if err != nil {
					onErrFn(fmt.Errorf("dialUDP: %s", err))

					s.Close()

					connsMux.Lock()
					delete(conns, udpaddr.String())
					connsMux.Unlock()
				}
			}
		}
	}()

	<-ctx.Done()
	conn.Close()

	onInfoFn("Closed " + addressstr)
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
