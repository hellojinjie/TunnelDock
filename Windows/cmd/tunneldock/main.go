package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

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
			app.ActivateExistingMainWindow("TunnelDock")
			return
		}
		log.Fatal(err)
	}
	defer instance.Close()

	walkApp, err := walk.InitApp()
	if err != nil {
		log.Fatal(err)
	}

	runtime, err := initializeRuntime()
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.job.Close()
	defer runtime.manager.Shutdown()

	mainWindow, err := ui.NewMainWindowWithConnector(runtime.model, runtime.manager)
	if err != nil {
		log.Fatal(err)
	}
	defer mainWindow.Dispose()
	dataRoot, err := tunnelDockDataRoot()
	if err != nil {
		log.Fatal(err)
	}
	trayController, err := app.NewTrayController(persistence.NewSettingsStore(filepath.Join(dataRoot, "settings.json")))
	if err != nil {
		log.Fatal(err)
	}
	quitting := false
	refresh := func() {
		go func() {
			_ = runtime.ReloadHosts(context.Background())
			mainWindow.RefreshHosts()
		}()
	}
	tray, err := ui.NewTray(mainWindow, trayController, runtime.manager, refresh, func() {
		quitting = true
		walk.App().Exit(0)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tray.Dispose()
	mainWindow.Closing().Attach(func(cancel *bool, _ walk.CloseReason) {
		if !quitting {
			tray.MinimizeOnClose(cancel)
		}
	})
	watchContext, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	go runtime.WatchConfig(watchContext, mainWindow.RefreshHosts)

	mainWindow.Show()
	walkApp.Run()
}

type runtime struct {
	model        *app.Model
	manager      *tunnel.Manager
	job          *sshclient.Job
	adapter      *tunnel.SSHProcessAdapter
	resolver     sshconfig.IncludeResolver
	hostResolver sshconfig.HostResolver
	configPath   string
	mu           sync.RWMutex
	expanded     sshconfig.ExpandedConfig
}

func initializeRuntime() (*runtime, error) {
	dataRoot, err := tunnelDockDataRoot()
	if err != nil {
		return nil, err
	}
	sshExecutable, err := sshclient.LocateOpenSSH()
	if err != nil {
		return nil, err
	}
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locate user home: %w", err)
	}
	sshDirectory := filepath.Join(homeDirectory, ".ssh")
	resolver := sshconfig.NewIncludeResolver(sshDirectory)
	expanded, err := resolver.Resolve(filepath.Join(sshDirectory, "config"))
	if err != nil {
		return nil, fmt.Errorf("read SSH configuration: %w", err)
	}
	hostResolver := sshconfig.NewHostResolver(sshExecutable, sshconfig.ExecRunner{})
	aliases := sshconfig.Scanner{}.DiscoverAliases(expanded.Lines)
	hosts := make([]model.SSHHost, 0, len(aliases))
	for order, alias := range aliases {
		hosts = append(hosts, hostResolver.Resolve(context.Background(), alias, order))
	}

	job, err := sshclient.NewJob()
	if err != nil {
		return nil, err
	}
	runtimeStore := sshconfig.NewRuntimeConfigStore(filepath.Join(dataRoot, "runtime"))
	if err := runtimeStore.RemoveStale(); err != nil {
		_ = job.Close()
		return nil, err
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
		return nil, err
	}
	manager.UpdateHosts(hosts)
	applicationModel := app.NewModel()
	applicationModel.SetHosts(hosts)
	return &runtime{
		model: applicationModel, manager: manager, job: job, adapter: processes,
		resolver: resolver, hostResolver: hostResolver, configPath: filepath.Join(sshDirectory, "config"), expanded: expanded,
	}, nil
}

// ReloadHosts re-resolves both the recursive SSH config and the OpenSSH
// effective values. New tunnels use the refreshed forwarding-sanitized input.
func (r *runtime) ReloadHosts(ctx context.Context) error {
	expanded, err := r.resolver.Resolve(r.configPath)
	if err != nil {
		return fmt.Errorf("read SSH configuration: %w", err)
	}
	aliases := sshconfig.Scanner{}.DiscoverAliases(expanded.Lines)
	hosts := make([]model.SSHHost, 0, len(aliases))
	for order, alias := range aliases {
		hosts = append(hosts, r.hostResolver.Resolve(ctx, alias, order))
	}
	r.adapter.SetExpandedConfig(expanded.Lines)
	r.manager.UpdateHosts(hosts)
	r.model.SetHosts(hosts)
	r.mu.Lock()
	r.expanded = expanded
	r.mu.Unlock()
	return nil
}

// WatchConfig resubscribes after every change so Include additions and glob
// changes are reflected without restarting TunnelDock.
func (r *runtime) WatchConfig(ctx context.Context, refreshed func()) {
	watcher := sshconfig.NewWatcher(300 * time.Millisecond)
	for {
		r.mu.RLock()
		expanded := r.expanded
		r.mu.RUnlock()
		events, err := watcher.Events(ctx, expanded)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		select {
		case <-ctx.Done():
			return
		case _, open := <-events:
			if !open {
				continue
			}
			_ = r.ReloadHosts(ctx)
			refreshed()
		}
	}
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
