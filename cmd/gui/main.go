package main

import (
	"errors"
	"sync"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
)

func main() {
	a := app.NewWithID("com.github.nickname32.p2p-forwarder")

	w := a.NewWindow("P2P Forwarder")
	w.Resize(fyne.NewSize(230, 280))

	connsWidget := createConnsWidget()

	logWidget, onInfoFn, onErrFn := createLogWidget()

	// FIXME
	onErrFn(errors.New("TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1TEST 1"))
	onInfoFn("TEST 2")

	w.SetContent(widget.NewVBox(connsWidget, logWidget))
	w.ShowAndRun()
}

func createConnsWidget() (connsWidget fyne.CanvasObject) {
	connections := widget.NewVBox()

	remoteidEntry := widget.NewEntry()
	remoteidEntry.SetPlaceHolder("Remote id")

	connectBtn := widget.NewButton("Connect", func() {
		text := remoteidEntry.Text
		println(text)
		hbox := widget.NewHBox()
		hbox.Append(widget.NewButton("disconnect", func() {
			for i := 0; i < len(connections.Children); i++ {
				if connections.Children[i] != hbox {
					continue
				}

				if i+1 < len(hbox.Children) {
					hbox.Children = append(hbox.Children[:i], hbox.Children[i+1:]...)
				} else {
					hbox.Children = append(hbox.Children[:i])
				}

				hbox.Refresh()
			}
		}))
		hbox.Append(widget.NewLabel(text))

		connections.Append(hbox)
	})

	return widget.NewVBox(widget.NewHBox(connectBtn, remoteidEntry), widget.NewScrollContainer(connections))
}

func createLogWidget() (logWidget fyne.CanvasObject, onInfoFn func(str string), onErrFn func(err error)) {
	logBox := widget.NewVBox()

	logScrollBox := widget.NewScrollContainer(logBox)
	logScrollBox.SetMinSize(fyne.Size{Height: 150})

	var logMux sync.Mutex
	addLogLine := func(msg string) {
		logMux.Lock()

		logBox.Append(widget.NewLabel(msg))
		logScrollBox.ScrollToBottom()

		i := len(logBox.Children) - 256
		if i < 0 {
			i = 0
		}
		logBox.Children = logBox.Children[i:]

		logMux.Unlock()
	}

	return logScrollBox, func(str string) {
			addLogLine(str)
		}, func(err error) {
			addLogLine(err.Error())
		}
}
