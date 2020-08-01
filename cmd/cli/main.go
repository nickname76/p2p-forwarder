package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	p2pforwarder "github.com/nickname32/p2p-forwarder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type strArrFlags []string

func (saf *strArrFlags) String() string {
	return fmt.Sprint([]string(*saf))
}

func (saf *strArrFlags) Set(value string) error {
	*saf = append(*saf, value)
	return nil
}

type uint16ArrFlags []uint16

func (uaf *uint16ArrFlags) String() string {
	return fmt.Sprint([]uint16(*uaf))
}

func (uaf *uint16ArrFlags) Set(value string) error {
	n, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return err
	}

	*uaf = append(*uaf, uint16(n))
	return nil
}

func main() {
	connectIds := strArrFlags{}
	flag.Var(&connectIds, "connect", "Add id you want connect to (can be used multiple times).")

	tcpPorts := uint16ArrFlags{}
	flag.Var(&tcpPorts, "tcp", "Add tcp port you want to open (can be used multiple times).")

	udpPorts := uint16ArrFlags{}
	flag.Var(&udpPorts, "udp", "Add udp port you want to open (can be used multiple times).")

	flag.Parse()

	p2pforwarder.OnError(func(err error) {
		zap.S().Error(err)
	})
	p2pforwarder.OnInfo(func(str string) {
		zap.L().Info(str)
	})

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

	zap.L().Info("Initialization...")

	fwr, cancel, err := p2pforwarder.NewForwarder()
	if err != nil {
		zap.S().Error(err)
	}

	cancels := make([]func(), 0)

	zap.L().Info("Your id: " + fwr.ID())

	for _, port := range tcpPorts {
		zap.L().Info("Opening tcp:" + strconv.FormatUint(uint64(port), 10))
		cancel, err := fwr.OpenPort("tcp", port)
		if err != nil {
			zap.S().Fatal(err)
		}
		cancels = append(cancels, cancel)
	}
	for _, port := range udpPorts {
		zap.L().Info("Opening udp:" + strconv.FormatUint(uint64(port), 10))
		cancel, err := fwr.OpenPort("udp", port)
		if err != nil {
			zap.S().Fatal(err)
		}
		cancels = append(cancels, cancel)
	}

	for _, id := range connectIds {
		zap.L().Info("Connecting to " + id)

		listenip, cancel, err := fwr.Connect(id)
		if err != nil {
			zap.S().Fatal(err)
		}
		cancels = append(cancels, cancel)

		zap.L().Info("Connections to " + id + "'s ports are listened on " + listenip)
	}

	cancels = append(cancels, cancel)

	zap.L().Info("Initialization completed")

	termSignal := make(chan os.Signal, 1)
	signal.Notify(termSignal, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-termSignal

	zap.L().Info("Shutdown...")

	for _, cancel := range cancels {
		cancel()
	}
}
