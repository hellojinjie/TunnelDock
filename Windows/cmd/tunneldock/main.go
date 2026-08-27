package main

import (
	"log"

	"github.com/tailscale/walk"
)

func main() {
	app, err := walk.InitApp()
	if err != nil {
		log.Fatal(err)
	}

	mainWindow, err := walk.NewMainWindow()
	if err != nil {
		log.Fatal(err)
	}
	defer mainWindow.Dispose()

	if err := mainWindow.SetTitle("TunnelDock"); err != nil {
		log.Fatal(err)
	}
	if err := mainWindow.SetSize(walk.Size{Width: 960, Height: 640}); err != nil {
		log.Fatal(err)
	}

	mainWindow.Show()
	app.Run()
}
