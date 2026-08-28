package ui

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/hellojinjie/TunnelDock/Windows/internal/app"
	"github.com/tailscale/walk"
)

var uiTestJobs chan func()
var uiTestInitErr error

func TestMain(m *testing.M) {
	runtime.LockOSThread()
	if _, err := walk.InitApp(); err != nil {
		uiTestInitErr = err
		os.Exit(m.Run())
	}
	uiTestJobs = make(chan func())
	done := make(chan int, 1)
	go func() { done <- m.Run() }()
	for {
		select {
		case job := <-uiTestJobs:
			job()
		case code := <-done:
			os.Exit(code)
		}
	}
}

type fixedAppearanceSource struct{ appearance Appearance }

func (s fixedAppearanceSource) Current() Appearance { return s.appearance }

func (s fixedAppearanceSource) Watch(ctx context.Context) <-chan Appearance {
	updates := make(chan Appearance)
	go func() {
		<-ctx.Done()
		close(updates)
	}()
	return updates
}

func TestMainWindowBuildsWithoutTableViews(t *testing.T) {
	if uiTestInitErr != nil {
		t.Skipf("Walk UI is unavailable in this test session: %v", uiTestInitErr)
	}
	var smokeErr error
	runOnTestUIThread(func() {
		env := newUIEnvironment(fixedAppearanceSource{appearance: AppearanceLight})
		defer env.Dispose()
		window, err := NewMainWindowWithEnvironment(app.NewModel(), nil, env)
		if err != nil {
			smokeErr = err
			return
		}
		defer window.Dispose()
		if containsWidgetType(window, "*walk.TableView") || containsWidgetType(window, "*walk.ListBox") {
			smokeErr = fmt.Errorf("main window still contains a table or list control")
		}
	})
	if smokeErr != nil {
		t.Fatal(smokeErr)
	}
}

func TestMainWindowSourceDoesNotReferenceNativeLists(t *testing.T) {
	source, err := os.ReadFile("main_window.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"walk.TableView", "walk.ListBox", "TableView{"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("main window references %q", forbidden)
		}
	}
}

func runOnTestUIThread(job func()) {
	done := make(chan struct{})
	uiTestJobs <- func() {
		defer close(done)
		job()
	}
	<-done
}

func containsWidgetType(window walk.Window, typeName string) bool {
	if reflect.TypeOf(window).String() == typeName {
		return true
	}
	container, ok := window.(walk.Container)
	if !ok {
		return false
	}
	for index := 0; index < container.Children().Len(); index++ {
		if containsWidgetType(container.Children().At(index), typeName) {
			return true
		}
	}
	return false
}
