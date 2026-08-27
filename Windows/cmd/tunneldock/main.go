package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
	"github.com/hellojinjie/TunnelDock/Windows/internal/persistence"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshclient"
	"github.com/hellojinjie/TunnelDock/Windows/internal/sshconfig"
	"github.com/hellojinjie/TunnelDock/Windows/internal/tunnel"
	"github.com/hellojinjie/TunnelDock/Windows/internal/ui"
	"github.com/tailscale/walk"
)

func main() {
	instance, err := app.AcquireSingleInstance("Local\\TunnelDock.Windows.Singleton")
	if err != nil {
		if errors.Is(err, app.ErrAlreadyRunning) {
			return
		}
		log.Fatal(err)
	}
	defer instance.Close()

	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatal(err)
	}

	applicationModel, manager, job, err := initializeRuntime()
	if err != nil {
		log.Fatal(err)
	}
	defer job.Close()
	defer manager.Shutdown()

	mainWindow, err := ui.NewMainWindowWithConnector(applicationModel, manager)
	if err != nil {
		log.Fatal(err)
	}
	defer mainWindow.Dispose()

	mainWindow.Show()
	walkApp.Run()
}

func initializeRuntime() (*app.Model, *tunnel.Manager, *sshclient.Job, error) {
	dataRoot, err := tunnelDockDataRoot()
	if err != nil {
		return nil, nil, nil, err
	}
	sshExecutable, err := sshclient.LocateOpenSSH()
	if err != nil {
		return nil, nil, nil, err
	}
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("locate user home: %w", err)
	}
	sshDirectory := filepath.Join(homeDirectory, ".ssh")
	expanded, err := sshconfig.NewIncludeResolver(sshDirectory).Resolve(filepath.Join(sshDirectory, "config"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read SSH configuration: %w", err)
	}
	hostResolver := sshconfig.NewHostResolver(sshExecutable, sshconfig.ExecRunner{})
	aliases := sshconfig.Scanner{}.DiscoverAliases(expanded.Lines)
	hosts := make([]model.SSHHost, 0, len(aliases))
	for order, alias := range aliases {
		hosts = append(hosts, hostResolver.Resolve(context.Background(), alias, order))
	}

	job, err := sshclient.NewJob()
	if err != nil {
		return nil, nil, nil, err
	}
	runtimeStore := sshconfig.NewRuntimeConfigStore(filepath.Join(dataRoot, "runtime"))
	if err := runtimeStore.RemoveStale(); err != nil {
		_ = job.Close()
		return nil, nil, nil, err
	}
	processes := tunnel.NewSSHProcessAdapter(
		sshclient.NewProcessController(runtimeStore, sshclient.ExecLauncher{}, job), sshExecutable, expanded.Lines,
	)
	manager := tunnel.NewManager(tunnel.ManagerOptions{
		Repository: persistence.NewTunnelRepository(filepath.Join(dataRoot, "saved-tunnels.json")),
		Ports:      tunnel.NewPortChecker(),
		Processes:  processes,
	})
	if err := manager.LoadSavedDefinitions(); err != nil {
		_ = job.Close()
		return nil, nil, nil, err
	}
	manager.UpdateHosts(hosts)
	applicationModel := app.NewModel()
	applicationModel.SetHosts(hosts)
	return applicationModel, manager, job, nil
}

func tunnelDockDataRoot() (string, error) {
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		return filepath.Join(localAppData, "TunnelDock"), nil
	}
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate application data directory: %w", err)
	}
	return filepath.Join(configDirectory, "TunnelDock"), nil
}
