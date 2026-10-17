// The composition, rendered without a display: `joucho --render FILE`
// draws the screen as the greeter lays it out — the paper canvas with the
// night panel over its right share, the card's footprint marked, and the
// anchors the concept's micro text is meant to sit on — so the
// arrangement can be checked and nudged without fighting a compositor.

package main

import (
	"math/rand/v2"

	"codeberg.org/puregotk/puregotk/v4/cairo"
)

// The screen the render stands in for, in logical pixels; a fullscreen
// greeter is this big on a 16:10 monitor with inua's bar out of the way.
const (
	renderW = 2380.0
	renderH = 1480.0
)

// The card's size in logical pixels, as assets/style.css gives it.
const (
	cardWpx = 400.0
	cardHpx = 300.0
)

// renderComposite draws the screen and saves it to path.
func (g *greeter) renderComposite(path string) error {
	const w, h = renderW, renderH
	g.layout(w)

	g.grain = newBitmap(240, 240)
	g.grain.Speckle(rand.New(rand.NewPCG(1, 2)), 7000, isInk, 0.055)
	g.panelGrain = newBitmap(240, 240)
	g.panelGrain.Speckle(rand.New(rand.NewPCG(3, 4)), 5200, isPaper, 0.08)

	im := newBitmap(int(w), int(h))
	defer im.Free()
	c := cr.create(im.s)
	ctx := (*cairo.Context)(c)
	p := newPen(ctx)

	// The paper ground, as the CSS gives it to the window.
	p.Rect(0, 0, w, h)
	p.Color(isPaper)
	p.Fill()

	g.drawDecor(p, w, h)

	// The anchors: where the concept's micro text is meant to sit.
	for _, a := range []struct{ x, y float64 }{{46, 38}, {54, 92}, {356, 272}, {820, 510}, {610, 120}} {
		p.Save()
		p.MoveTo(a.x-12, a.y)
		p.LineTo(a.x+12, a.y)
		p.MoveTo(a.x, a.y-12)
		p.LineTo(a.x, a.y+12)
		p.Width(1)
		p.Color(isSun)
		p.Stroke()
		p.Restore()
	}

	// The card's footprint, where the overlay puts it: centred over the
	// paper, a little left of the paper's middle.
	paperW := w - g.panelW
	cx := 0.36 * paperW
	p.Save()
	p.Rect(cx-cardWpx/2, h/2-cardHpx/2, cardWpx, cardHpx)
	p.Alpha(isRust, 0.10)
	p.Fill()
	p.Rect(cx-cardWpx/2, h/2-cardHpx/2, cardWpx, cardHpx)
	p.Width(1)
	p.Color(isRust)
	p.Stroke()
	p.Restore()

	// The night panel over the stage's right share, as its box does.
	p.Rect(w-g.panelW, 0, g.panelW, h)
	p.Color(isInk)
	p.Fill()
	p.Save()
	p.Translate(w-g.panelW, 0)
	g.drawPanel(p, g.panelW, h)
	p.Restore()

	cr.destroy(ctx.GoPointer())
	cr.flushSurface(im.s)
	return im.SavePNG(path)
}
