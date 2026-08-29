package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/tailscale/walk"
)

type resourceKey struct {
	appearance Appearance
	dpi        int
}

type UIResources struct {
	Appearance    Appearance
	DPI           int
	Palette       Palette
	Metrics       Metrics
	BodyFont      *walk.Font
	MediumFont    *walk.Font
	TitleFont     *walk.Font
	CaptionFont   *walk.Font
	MonoFont      *walk.Font
	WindowBrush   *walk.SolidColorBrush
	SidebarBrush  *walk.SolidColorBrush
	SurfaceBrush  *walk.SolidColorBrush
	HoverBrush    *walk.SolidColorBrush
	SelectedBrush *walk.SolidColorBrush
	BorderPen     *walk.CosmeticPen
	FocusPen      *walk.CosmeticPen
	iconBitmaps   map[iconBitmapKey]*walk.Bitmap
}

func newUIResources(appearance Appearance, dpi int) (*UIResources, error) {
	resources := &UIResources{Appearance: appearance, DPI: dpi, Palette: PaletteFor(appearance), Metrics: MetricsForDPI(dpi), iconBitmaps: make(map[iconBitmapKey]*walk.Bitmap)}
	var err error
	if resources.BodyFont, err = walk.NewFont("Segoe UI", 11, walk.FontNormal); err != nil {
		resources.Dispose()
		return nil, err
	}
	if resources.MediumFont, err = walk.NewFont("Segoe UI", 11, walk.FontBold); err != nil {
		resources.Dispose()
		return nil, err
	}
	if resources.TitleFont, err = walk.NewFont("Segoe UI", 24, walk.FontBold); err != nil {
		resources.Dispose()
		return nil, err
	}
	if resources.CaptionFont, err = walk.NewFont("Segoe UI", 9, walk.FontNormal); err != nil {
		resources.Dispose()
		return nil, err
	}
	if resources.MonoFont, err = walk.NewFont("Cascadia Mono", 10, walk.FontNormal); err != nil {
		resources.Dispose()
		return nil, err
	}
	brushes := []struct {
		target **walk.SolidColorBrush
		color  walk.Color
	}{
		{&resources.WindowBrush, resources.Palette.Window},
		{&resources.SidebarBrush, resources.Palette.Sidebar},
		{&resources.SurfaceBrush, resources.Palette.Surface},
		{&resources.HoverBrush, resources.Palette.SurfaceHover},
		{&resources.SelectedBrush, resources.Palette.SurfaceSelected},
	}
	for _, item := range brushes {
		if *item.target, err = walk.NewSolidColorBrush(item.color); err != nil {
			resources.Dispose()
			return nil, err
		}
	}
	if resources.BorderPen, err = walk.NewCosmeticPen(walk.PenSolid, resources.Palette.Border); err != nil {
		resources.Dispose()
		return nil, err
	}
	if resources.FocusPen, err = walk.NewCosmeticPen(walk.PenSolid, resources.Palette.Focus); err != nil {
		resources.Dispose()
		return nil, err
	}
	return resources, nil
}

func (r *UIResources) Dispose() {
	if r == nil {
		return
	}
	for _, font := range []*walk.Font{r.BodyFont, r.MediumFont, r.TitleFont, r.CaptionFont, r.MonoFont} {
		if font != nil {
			font.Dispose()
		}
	}
	for _, brush := range []*walk.SolidColorBrush{r.WindowBrush, r.SidebarBrush, r.SurfaceBrush, r.HoverBrush, r.SelectedBrush} {
		if brush != nil {
			brush.Dispose()
		}
	}
	for _, pen := range []*walk.CosmeticPen{r.BorderPen, r.FocusPen} {
		if pen != nil {
			pen.Dispose()
		}
	}
	for _, bitmap := range r.iconBitmaps {
		bitmap.Dispose()
	}
	r.iconBitmaps = nil
}

type UIEnvironment struct {
	mu          sync.RWMutex
	source      appearanceSource
	appearance  Appearance
	resources   map[resourceKey]*UIResources
	subscribers map[int]func(Appearance)
	nextID      int
	cancel      context.CancelFunc
	done        chan struct{}
	disposed    bool
	synchronize func(func())
}

func NewUIEnvironment() (*UIEnvironment, error) {
	return newUIEnvironment(windowsAppearanceSource{}), nil
}

func newUIEnvironment(source appearanceSource) *UIEnvironment {
	return newUIEnvironmentWithSynchronizer(source, func(callback func()) {
		walk.App().Synchronize(callback)
	})
}

func newUIEnvironmentWithSynchronizer(source appearanceSource, synchronize func(func())) *UIEnvironment {
	ctx, cancel := context.WithCancel(context.Background())
	environment := &UIEnvironment{
		source: source, appearance: source.Current(), resources: make(map[resourceKey]*UIResources),
		subscribers: make(map[int]func(Appearance)), cancel: cancel, done: make(chan struct{}), synchronize: synchronize,
	}
	go environment.watch(ctx)
	return environment
}

func (e *UIEnvironment) Appearance() Appearance {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.appearance
}

func (e *UIEnvironment) Resources(dpi int) (*UIResources, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.disposed {
		return nil, fmt.Errorf("UI environment is disposed")
	}
	key := resourceKey{appearance: e.appearance, dpi: dpi}
	if resources := e.resources[key]; resources != nil {
		return resources, nil
	}
	resources, err := newUIResources(e.appearance, dpi)
	if err != nil {
		return nil, err
	}
	e.resources[key] = resources
	return resources, nil
}

func (e *UIEnvironment) ApplyNativeFont(window walk.Window, dpi int) error {
	resources, err := e.Resources(dpi)
	if err != nil {
		return err
	}
	applyTextScale(window, resources.BodyFont)
	return nil
}

func applyTextScale(window walk.Window, font *walk.Font) {
	window.SetFont(font)
	container, isContainer := window.(walk.Container)
	if !isContainer {
		return
	}
	children := container.Children()
	for index := 0; index < children.Len(); index++ {
		applyTextScale(children.At(index), font)
	}
}

func (e *UIEnvironment) Subscribe(callback func(Appearance)) func() {
	e.mu.Lock()
	if e.disposed {
		e.mu.Unlock()
		return func() {}
	}
	id := e.nextID
	e.nextID++
	e.subscribers[id] = callback
	e.mu.Unlock()
	return func() {
		e.mu.Lock()
		delete(e.subscribers, id)
		e.mu.Unlock()
	}
}

func (e *UIEnvironment) watch(ctx context.Context) {
	defer close(e.done)
	for appearance := range e.source.Watch(ctx) {
		e.mu.Lock()
		if e.disposed || appearance == e.appearance {
			e.mu.Unlock()
			continue
		}
		e.appearance = appearance
		subscriberIDs := make([]int, 0, len(e.subscribers))
		for id := range e.subscribers {
			subscriberIDs = append(subscriberIDs, id)
		}
		e.mu.Unlock()
		e.synchronize(func() {
			e.mu.RLock()
			if e.disposed || e.appearance != appearance {
				e.mu.RUnlock()
				return
			}
			callbacks := make([]func(Appearance), 0, len(subscriberIDs))
			for _, id := range subscriberIDs {
				if callback := e.subscribers[id]; callback != nil {
					callbacks = append(callbacks, callback)
				}
			}
			e.mu.RUnlock()
			for _, callback := range callbacks {
				callback(appearance)
			}
		})
	}
}

func (e *UIEnvironment) Dispose() {
	if e == nil {
		return
	}
	e.cancel()
	<-e.done
	e.mu.Lock()
	if e.disposed {
		e.mu.Unlock()
		return
	}
	e.disposed = true
	resources := e.resources
	e.resources = nil
	e.subscribers = nil
	e.mu.Unlock()
	for _, item := range resources {
		item.Dispose()
	}
}
