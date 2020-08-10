package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	p2pforwarder "github.com/nickname32/p2p-forwarder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	logger, err := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:     "Time",
			LevelKey:    "Level",
			MessageKey:  "Message",
			LineEnding:  zapcore.DefaultLineEnding,
			EncodeLevel: zapcore.LowercaseLevelEncoder,
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(t.Format("2006/01/02 15:04:05.000"))
			},
			EncodeDuration: nil,
			EncodeCaller:   nil,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(logger)

	p2pforwarder.OnError(func(err error) {
		zap.S().Error(err)
	})
	p2pforwarder.OnInfo(func(str string) {
		zap.L().Info(str)
	})
}

type strArrFlags []string

func (saf *strArrFlags) String() string {
	return fmt.Sprint([]string(*saf))
}

func (saf *strArrFlags) Set(value string) error {
	*saf = append(*saf, value)
	return nil
}

var (
	fwr          *p2pforwarder.Forwarder
	fwrCancel    func()
	connections  = make(map[string]func())
	openTCPPorts = make(map[uint16]func())
	openUDPPorts = make(map[uint16]func())
)

func main() {
	connectIds := strArrFlags{}
	flag.Var(&connectIds, "connect", "Add id you want connect to (can be used multiple times).")

	tcpPorts := strArrFlags{}
	flag.Var(&tcpPorts, "tcp", "Add tcp port you want to open (can be used multiple times).")

	udpPorts := strArrFlags{}
	flag.Var(&udpPorts, "udp", "Add udp port you want to open (can be used multiple times).")

	flag.Parse()

	zap.L().Info("Initialization...")

	var err error

	fwr, fwrCancel, err = p2pforwarder.NewForwarder()
	if err != nil {
		zap.S().Error(err)
	}

	zap.L().Info("Your id: " + fwr.ID())

	for _, port := range tcpPorts {
		cmdOpen([]string{"tcp", port})
	}
	for _, port := range udpPorts {
		cmdOpen([]string{"udp", port})
	}

	for _, id := range connectIds {
		cmdConnect([]string{id})
	}

	zap.L().Info("Initialization completed")

	cmdch := make(chan string)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		for {
			scanner.Scan()
			err = scanner.Err()

			if err != nil {
				zap.S().Error(err)
				continue
			}

			cmdch <- scanner.Text()
		}
	}()

	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	executeCommand("")

loop:
	for {
		select {
		case str := <-cmdch:
			executeCommand(str)
		case <-termSignal:
			shutdown()
			break loop
		}
	}
}
func shutdown() {
	zap.L().Info("Shutdown...")

	fwrCancel()

	for _, cancel := range connections {
		cancel()
	}
	for _, cancel := range openTCPPorts {
		cancel()
	}
	for _, cancel := range openUDPPorts {
		cancel()
	}
}

func parseArgs(argsStr string, n int) []string {
	if n <= 0 {
		return []string{}
	}

	args := make([]string, n)

	argsRunes := []rune(argsStr)

	argStart := -1

	a := 0

	for i := 0; i < len(argsRunes); i++ {
		if argStart == -1 {
			if unicode.IsSpace(argsRunes[i]) {
				continue
			} else {
				argStart = i
			}
		} else {
			if a+1 == n {
				args[a] = string(argsRunes[argStart:])
				break
			} else {
				if unicode.IsSpace(argsRunes[i]) {
					args[a] = string(argsRunes[argStart:i])
					argStart = -1
					a++
				}
			}
		}
	}

	if argStart != -1 {
		args[a] = string(argsRunes[argStart:])
	}

	return args
}

func executeCommand(str string) {
	args := parseArgs(str, 3)
	cmd := strings.ToLower(args[0])
	params := args[1:]

	switch cmd {
	case "connect":
		cmdConnect(params)
	case "disconnect":
		cmdDisconnect(params)
	case "open":
		cmdOpen(params)
	case "close":
		cmdClose(params)
	default:
		zap.L().Info("")
		zap.L().Info("Cli commands list:")
		zap.L().Info("connect [ID_HERE]")
		zap.L().Info("disconnect [ID_HERE]")
		zap.L().Info("open [TCP_OR_UDP_HERE] [PORT_NUMBER_HERE]")
		zap.L().Info("close [UDP_OR_UDP_HERE] [PORT_NUMBER_HERE]")
		zap.L().Info("")
	}
}

func cmdConnect(params []string) {
	id := params[0]

	zap.L().Info("Connecting to " + id)

	listenip, cancel, err := fwr.Connect(id)
	if err != nil {
		zap.S().Error(err)
		return
	}

	connections[id] = cancel

	zap.L().Info("Connections to " + id + "'s ports are listened on " + listenip)
}

func cmdDisconnect(params []string) {
	id := params[0]

	close := connections[id]

	if close == nil {
		zap.L().Error("You are not connected to specified id")
		return
	}

	zap.L().Info("Disconnecting from " + id)

	close()

	delete(connections, id)
}

func cmdOpen(params []string) {
	networkType := strings.ToLower(params[0])

	portStr := params[1]
	portUint64, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		zap.S().Error(err)
		return
	}
	port := uint16(portUint64)

	zap.L().Info("Opening " + networkType + ":" + portStr)

	cancel, err := fwr.OpenPort(networkType, port)
	if err != nil {
		zap.S().Error(err)
		return
	}

	switch networkType {
	case "tcp":
		openTCPPorts[port] = cancel
	case "udp":
		openUDPPorts[port] = cancel
	}
}

func cmdClose(params []string) {
	networkType := strings.ToLower(params[0])
	portStr := params[1]
	portUint64, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		zap.S().Error(err)
		return
	}
	port := uint16(portUint64)

	var cancel func()
	switch networkType {
	case "tcp":
		cancel = openTCPPorts[port]
	case "udp":
		cancel = openUDPPorts[port]
	}

	if cancel == nil {
		zap.L().Error("Specified port is not opened")
		return
	}

	zap.L().Info("Closing " + networkType + ":" + portStr)

	cancel()

	delete(openTCPPorts, port)
}
