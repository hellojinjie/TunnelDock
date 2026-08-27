package main

import (
	"log"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/ui"
	"github.com/tailscale/walk"
)

func main() {
	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatal(err)
	}

	mainWindow, err := ui.NewMainWindow(app.NewModel())
	if err != nil {
		log.Fatal(err)
	}
	defer mainWindow.Dispose()

	mainWindow.Show()
	walkApp.Run()
}
