package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/VladimirMarkelov/clui"
	p2pforwarder "github.com/nickname32/p2p-forwarder"
)

func main() {
	clui.InitLibrary()
	defer clui.DeinitLibrary()

	createView()

	clui.MainLoop()
}

func createView() {
	win := clui.AddWindow(0, 0, 0, 0, "P2P Forwarder")
	win.SetPack(clui.Horizontal)
	win.SetTitleButtons(clui.ButtonDefault)
	win.SetMaximized(true)
	win.SetSizable(false)
	win.SetMovable(false)
	win.SetBorder(clui.BorderNone)

	frame := clui.CreateFrame(win, 0, 0, clui.BorderThin, clui.AutoSize)
	frame.SetPack(clui.Vertical)

	onErrFn, onInfoFn := createLog(clui.CreateFrame(win, 0, 0, clui.BorderThin, clui.AutoSize))
	p2pforwarder.OnError(onErrFn)
	p2pforwarder.OnInfo(onInfoFn)

	label := clui.CreateLabel(frame, 64, 1, "Initialization...", clui.AutoSize)
	clui.RefreshScreen()

	fwr, cancel, err := p2pforwarder.NewForwarder()
	if err != nil {
		label.SetTitle("Error: " + err.Error())
		return
	}

	win.OnClose(func(_ clui.Event) bool {
		onInfoFn("Shutting down...")
		cancel()
		return true
	})

	label.Destroy()

	createYourID(frame, fwr)
	createConnections(frame, fwr)
	createPortsControl(frame, fwr)
}

func createLog(parent clui.Control) (onErrFn func(error), onInfoFn func(string)) {
	textView := clui.CreateTextView(parent, 0, 0, clui.AutoSize)
	textView.SetMaxItems(500)
	textView.SetAutoScroll(true)

	logCh := make(chan string, 2)
	go func() {
		for {
			msg := <-logCh
			textView.AddText([]string{time.Now().Format("15:04:05.000") + " " + msg})
			clui.RefreshScreen()
		}
	}()

	return func(err error) {
			logCh <- "Error - " + err.Error()
		}, func(str string) {
			logCh <- "Info - " + str
		}
}

func createYourID(parent clui.Control, fwr *p2pforwarder.Forwarder) {
	frame := clui.CreateFrame(parent, 0, 0, clui.BorderThick, clui.Fixed)

	clui.CreateLabel(frame, 8, 1, "Your ID", clui.Fixed)

	editField := clui.CreateEditField(frame, 56, fwr.ID(), clui.Fixed)
	editField.OnChange(func(_ clui.Event) {
		editField.SetTitle(fwr.ID())
	})
}

func createConnections(parent clui.Control, fwr *p2pforwarder.Forwarder) {
	clui.CreateLabel(clui.CreateFrame(parent, 0, 0, clui.BorderThin, clui.Fixed), 11, 1, "Connections", clui.Fixed)

	frameA := clui.CreateFrame(parent, 0, 0, clui.BorderNone, clui.Fixed)
	frameA.SetPack(clui.Horizontal)

	buttonA := clui.CreateButton(frameA, 9, 4, "Connect", clui.Fixed)

	frameB := clui.CreateFrame(frameA, 0, 0, clui.BorderNone, clui.Fixed)
	frameB.SetPack(clui.Vertical)

	editField := clui.CreateEditField(frameB, 56, "id here", clui.Fixed)

	frameC := clui.CreateFrame(frameB, 0, 0, clui.BorderNone, clui.Fixed)
	frameC.SetPack(clui.Horizontal)

	label := clui.CreateLabel(frameB, 56, 1, "", clui.Fixed)

	frameD := clui.CreateFrame(parent, 0, 0, clui.BorderNone, clui.Fixed)
	frameA.SetPack(clui.Horizontal)
	buttonB := clui.CreateButton(frameD, 9, 4, "Disconn", clui.Fixed)
	listBox := clui.CreateListBox(frameD, 56, 4, clui.Fixed)

	connsMap := map[string]func(){}

	buttonA.OnClick(func(_ clui.Event) {
		connInfo := strings.TrimSpace(editField.Title())

		listenip, cancel, err := fwr.Connect(connInfo)
		if err != nil {
			label.SetTitle("Error: " + err.Error())
			return
		}

		ok := listBox.AddItem(connInfo)
		if !ok {
			clui.RefreshScreen()
			ok = listBox.AddItem(connInfo)
			if !ok {
				panic(ok)
			}
		}

		connsMap[connInfo] = cancel

		label.SetTitle("Connections are listened on " + listenip)
	})
	buttonB.OnClick(func(_ clui.Event) {
		itemid := listBox.SelectedItem()
		if itemid == -1 {
			return
		}

		connInfo, ok := listBox.Item(itemid)
		if !ok {
			return
		}

		ok = listBox.RemoveItem(itemid)
		if !ok {
			listBox.SelectItem(0)
			clui.RefreshScreen()
			return
		}

		connsMap[connInfo]()

		delete(connsMap, connInfo)
	})
}

func createPortsControl(parent clui.Control, fwr *p2pforwarder.Forwarder) {
	clui.CreateLabel(clui.CreateFrame(parent, 0, 0, clui.BorderThin, clui.Fixed), 13, 1, "Ports control", clui.Fixed)

	frameA := clui.CreateFrame(parent, 0, 0, clui.BorderNone, clui.Fixed)
	frameA.SetPack(clui.Horizontal)

	buttonA := clui.CreateButton(frameA, 9, 4, "Open", clui.Fixed)

	frameB := clui.CreateFrame(frameA, 0, 0, clui.BorderNone, clui.Fixed)
	frameB.SetPack(clui.Vertical)

	frameC := clui.CreateFrame(frameB, 0, 0, clui.BorderNone, clui.Fixed)
	frameC.SetPack(clui.Horizontal)

	editFieldA := clui.CreateEditField(frameC, 13, "tcp/udp here", clui.Fixed)
	clui.CreateLabel(frameC, 1, 1, " ", clui.Fixed)
	editFieldB := clui.CreateEditField(frameC, 16, "port here", clui.Fixed)

	label := clui.CreateLabel(frameB, 56, 1, "", clui.Fixed)

	frameD := clui.CreateFrame(parent, 0, 0, clui.BorderNone, clui.Fixed)
	frameA.SetPack(clui.Horizontal)
	buttonB := clui.CreateButton(frameD, 9, 4, "Close", clui.Fixed)
	listBox := clui.CreateListBox(frameD, 56, 4, clui.Fixed)

	portsMap := map[string]func(){}

	buttonA.OnClick(func(_ clui.Event) {
		portstr := strings.TrimSpace(editFieldB.Title())
		networkType := strings.ToLower(strings.TrimSpace(editFieldA.Title()))

		port, err := strconv.ParseUint(portstr, 10, 16)
		if err != nil {
			label.SetTitle("Error: " + err.Error())
			return
		}

		cancel, err := fwr.OpenPort(networkType, uint16(port))
		if err != nil {
			label.SetTitle("Error: " + err.Error())
			return
		}

		portInfo := networkType + ":" + portstr

		ok := listBox.AddItem(portInfo)
		if !ok {
			clui.RefreshScreen()
			ok = listBox.AddItem(portInfo)
			if !ok {
				panic(ok)
			}
		}

		portsMap[portInfo] = cancel

		label.SetTitle("Port " + portInfo + " opened")
	})
	buttonB.OnClick(func(_ clui.Event) {
		itemid := listBox.SelectedItem()
		if itemid == -1 {
			return
		}

		portInfo, ok := listBox.Item(itemid)
		if !ok {
			return
		}

		ok = listBox.RemoveItem(itemid)
		if !ok {
			listBox.SelectItem(0)
			clui.RefreshScreen()
			return
		}

		portsMap[portInfo]()

		delete(portsMap, portInfo)
	})
}
