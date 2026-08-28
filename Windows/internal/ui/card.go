package ui

import "github.com/tailscale/walk"

// Card groups native controls on the same semantic surface used by custom
// rows. The outer container owns spacing; Content is the only child callers
// should populate.
type Card struct {
	*walk.Composite
	Content     *walk.Composite
	env         *UIEnvironment
	unsubscribe func()
}

func NewCard(parent walk.Container, env *UIEnvironment) (*Card, error) {
	outer, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	card := &Card{Composite: outer, env: env}
	fail := func(cause error) (*Card, error) {
		outer.Dispose()
		return nil, cause
	}
	outerLayout := walk.NewVBoxLayout()
	outerLayout.SetMargins(walk.Margins{HNear: 1, VNear: 1, HFar: 1, VFar: 1})
	if err := outer.SetLayout(outerLayout); err != nil {
		return fail(err)
	}
	resources, err := env.Resources(outer.DPI())
	if err != nil {
		return fail(err)
	}
	outer.SetBackground(resources.SurfaceBrush)
	card.Content, err = walk.NewComposite(outer)
	if err != nil {
		return fail(err)
	}
	card.Content.SetBackground(resources.SurfaceBrush)
	card.unsubscribe = env.Subscribe(func(Appearance) {
		if refreshed, resourceErr := env.Resources(card.DPI()); resourceErr == nil {
			card.SetBackground(refreshed.SurfaceBrush)
			card.Content.SetBackground(refreshed.SurfaceBrush)
			card.Invalidate()
		}
	})
	return card, nil
}

func (c *Card) Dispose() {
	if c.unsubscribe != nil {
		c.unsubscribe()
		c.unsubscribe = nil
	}
	c.Composite.Dispose()
}
