// ===== UNDER DEVELOPMENT =====

package main

import (
	"os"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var application *gtk.Application

func main() {
	// Create Gtk Application, change appID to your application domain name reversed.
	const appID = "com.github.nickname32.p2p-forwarder"
	var err error
	application, err = gtk.ApplicationNew(appID, glib.APPLICATION_FLAGS_NONE)
	// Check to make sure no errors when creating Gtk Application
	if err != nil {
		panic(err)
	}
	// Application signals available
	// startup -> sets up the application when it first starts
	// activate -> shows the default first window of the application (like a new document). This corresponds to the application being launched by the desktop environment.
	// open -> opens files and shows them in a new window. This corresponds to someone trying to open a document (or documents) using the application from the file browser, or similar.
	// shutdown ->  performs shutdown tasks
	// Setup Gtk Application callback signals
	application.Connect("activate", func() {
		// Create ApplicationWindow
		appWindow, err := gtk.ApplicationWindowNew(application)
		if err != nil {
			panic(err)
		}
		// Set ApplicationWindow Properties
		appWindow.SetTitle("P2P Forwarder")
		appWindow.SetDefaultSize(540, 240)

		appWindow.Show()

		go start(appWindow)
	})
	// Run Gtk application
	os.Exit(application.Run(os.Args))
}

func start(win *gtk.ApplicationWindow) {
	label, err := gtk.LabelNew("Initialization...")
	if err != nil {
		panic(err)
	}
	label.Show()

	win.Add(label)

	time.Sleep(time.Second)

	label.Destroy()

	application.Connect("shutdown", func() {
		// TODO: shutdown of everything
	})

	box, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		panic(err)
	}

	yourid := createYourid("id_here")
	box.Add(yourid)

	logWidget, onErrFn, onInfoFn := createLog()
	box.Add(logWidget)
	_ = onErrFn
	_ = onInfoFn

	win.Add(box)

	win.ShowAll()
}

func createYourid(id string) gtk.IWidget {
	frame, err := gtk.FrameNew("Your id")
	if err != nil {
		panic(err)
	}

	entryBuf, err := gtk.EntryBufferNew(id, len(id))
	if err != nil {
		panic(err)
	}

	entry, err := gtk.EntryNewWithBuffer(entryBuf)
	if err != nil {
		panic(err)
	}

	frame.Add(entry)

	return frame
}

func createConnections() gtk.IWidget {
	box, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		panic(err)
	}

	button, err := gtk.ButtonNewWithLabel("Connect")
	if err != nil {
		panic(err)
	}
	box.Add(button)

	id := "id here"

	entryBuf, err := gtk.EntryBufferNew(id, len(id))
	if err != nil {
		panic(err)
	}

	entry, err := gtk.EntryNewWithBuffer(entryBuf)
	if err != nil {
		panic(err)
	}
	box.Add(entry)

	button.Connect("clicked", func() {
		println("Connect")
	})

	return nil
}

func createLog() (gtk.IWidget, func(error), func(string)) {
	// TODO: log
	label, err := gtk.LabelNew("TODO: log")
	if err != nil {
		panic(err)
	}

	return label, func(err error) {
			println(time.Now().Format("15:04:05.000") + " Error " + err.Error())
		}, func(str string) {
			println(time.Now().Format("15:04:05.000") + " Info " + str)
		}
}
