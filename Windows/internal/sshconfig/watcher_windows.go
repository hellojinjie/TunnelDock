package sshconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

const defaultWatchDebounce = 300 * time.Millisecond

type Watcher struct {
	debounce time.Duration
}

func NewWatcher(debounce time.Duration) Watcher {
	if debounce <= 0 {
		debounce = defaultWatchDebounce
	}
	return Watcher{debounce: debounce}
}

func (w Watcher) Events(ctx context.Context, config ExpandedConfig) (<-chan struct{}, error) {
	directories := watchDirectories(config)
	type subscription struct {
		handle windows.Handle
		path   string
	}
	subscriptions := make([]subscription, 0, len(directories))
	filter := uint32(windows.FILE_NOTIFY_CHANGE_FILE_NAME | windows.FILE_NOTIFY_CHANGE_DIR_NAME |
		windows.FILE_NOTIFY_CHANGE_LAST_WRITE | windows.FILE_NOTIFY_CHANGE_SIZE | windows.FILE_NOTIFY_CHANGE_CREATION)
	for _, directory := range directories {
		handle, err := windows.FindFirstChangeNotification(directory, false, filter)
		if err != nil {
			for _, existing := range subscriptions {
				_ = windows.FindCloseChangeNotification(existing.handle)
			}
			return nil, fmt.Errorf("watch SSH config directory %q: %w", directory, err)
		}
		subscriptions = append(subscriptions, subscription{handle: handle, path: directory})
	}

	raw := make(chan struct{}, 32)
	var waitGroup sync.WaitGroup
	for _, current := range subscriptions {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			defer windows.FindCloseChangeNotification(current.handle)
			watchChangeHandle(ctx, current.handle, raw)
		}()
	}
	go func() { waitGroup.Wait(); close(raw) }()
	return debounceEvents(ctx, raw, w.debounce), nil
}

func watchChangeHandle(ctx context.Context, handle windows.Handle, events chan<- struct{}) {
	for {
		result, err := windows.WaitForSingleObject(handle, 100)
		if err != nil {
			return
		}
		switch result {
		case windows.WAIT_OBJECT_0:
			select {
			case events <- struct{}{}:
			default:
			}
			if err := windows.FindNextChangeNotification(handle); err != nil {
				return
			}
		case uint32(windows.WAIT_TIMEOUT):
			select {
			case <-ctx.Done():
				return
			default:
			}
		default:
			return
		}
	}
}

func debounceEvents(ctx context.Context, input <-chan struct{}, delay time.Duration) <-chan struct{} {
	output := make(chan struct{}, 1)
	go func() {
		defer close(output)
		var timer *time.Timer
		var timerChannel <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case _, open := <-input:
				if !open {
					if timerChannel == nil {
						return
					}
					input = nil
					continue
				}
				if timer == nil {
					timer = time.NewTimer(delay)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(delay)
				}
				timerChannel = timer.C
			case <-timerChannel:
				select {
				case output <- struct{}{}:
				default:
				}
				timerChannel = nil
				if input == nil {
					return
				}
			}
		}
	}()
	return output
}

func watchDirectories(config ExpandedConfig) []string {
	candidates := append([]string(nil), config.WatchedDirectories...)
	for _, file := range config.WatchedFiles {
		candidates = append(candidates, filepath.Dir(file))
	}
	unique := make(map[string]struct{})
	for _, candidate := range candidates {
		if existing := nearestExistingDirectory(candidate); existing != "" {
			unique[existing] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for directory := range unique {
		result = append(result, directory)
	}
	sort.Strings(result)
	return result
}

func nearestExistingDirectory(path string) string {
	path = filepath.Clean(path)
	for {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
}
