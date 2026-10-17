// The isekai login screen: the ISEKAI JŌCHO concept as a greeter, with its
// permission. Paper and ink, a dark night panel on the right, the big ヰ,
// vines and a rust thread, the eye-flower, vertical writing, micro labels,
// the palette, and paper grain. Drawn with cairo in a GtkDrawingArea, in
// the concept's 1600 × 1000 units (the paper in 1080, the night panel in
// 520), scaled to whatever the screen is. Theme "isekai" picks it; every
// other theme draws the plain greeter.

package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"

	"codeberg.org/puregotk/puregotk/v4/cairo"
	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
)

// The concept's five colours.
var (
	isPaper = color.NRGBA{0xF4, 0xF0, 0xE8, 0xFF}
	isInk   = color.NRGBA{0x14, 0x13, 0x1A, 0xFF}
	isRust  = color.NRGBA{0xA6, 0x40, 0x2B, 0xFF}
	isEye   = color.NRGBA{0x7F, 0x8C, 0x94, 0xFF}
	isSun   = color.NRGBA{0xF2, 0xC2, 0x30, 0xFF}
)

// The concept's type: a Mincho for the display, a grotesque for the
// micro labels (its own stacks, with what Linux has).
const (
	fontSerifJP  = "Noto Serif CJK JP"
	fontSerifLat = "Noto Serif"
	fontSans     = "Noto Sans"
)

// fontPx is a Pango description for a size given in pixels (Pango wants
// points; 96 dpi, as everywhere else).
func fontPx(family string, px float64) string {
	return fmt.Sprintf("%s %.2f", family, px*0.75)
}

// fade is c at a fraction of its opacity, for the concept's quiet lines.
func fade(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(a*255 + 0.5)
	return c
}

// isekai reports whether the concept's dress is on.
func (g *greeter) isekai() bool { return g.cfg.Theme == "isekai" }

// buildIsekai fills win with the concept: the 1600 × 1000 canvas drawn
// over the whole window, the night panel lying over its right 520 units as
// in the drawing, and the card on the paper, left of the mark.
func (g *greeter) buildIsekai(win *gtk.ApplicationWindow) {
	g.grain = newBitmap(240, 240)
	g.grain.Speckle(rand.New(rand.NewPCG(1, 2)), 7000, isInk, 0.055)
	g.panelGrain = newBitmap(240, 240)
	g.panelGrain.Speckle(rand.New(rand.NewPCG(3, 4)), 5200, isPaper, 0.08)

	g.panelW = math.Round(panelShare * g.width())

	root := gtk.NewOverlay()

	stage := gtk.NewDrawingArea()
	stage.SetHexpand(true)
	stage.SetVexpand(true)
	g.onDecorDraw = func(_ uintptr, ctx *cairo.Context, width, height int32, _ uintptr) {
		g.drawDecor(newPen(ctx), float64(width), float64(height))
	}
	stage.SetDrawFunc(&g.onDecorDraw, 0, &noDestroy)
	root.SetChild(&stage.Widget)
	stage.Unref()

	// The card on the paper, its centre a little left of the paper's.
	g.card = g.buildCard()
	card := g.card
	card.SetHalign(gtk.AlignCenterValue)
	card.SetValign(gtk.AlignCenterValue)
	paperW := g.width() - g.panelW
	card.SetMarginEnd(int32(math.Round(g.width() - 0.72*paperW)))
	root.AddOverlay(&card.Widget)
	card.Unref()

	g.panel = gtk.NewBox(gtk.OrientationVerticalValue, 0)
	panel := g.panel
	panel.AddCssClass("isekai-night")
	panel.SetSizeRequest(int32(g.panelW), -1)
	panel.SetHalign(gtk.AlignEndValue)
	panel.SetValign(gtk.AlignFillValue)
	night := gtk.NewDrawingArea()
	night.SetHexpand(true)
	night.SetVexpand(true)
	g.onPanelDraw = func(_ uintptr, ctx *cairo.Context, width, height int32, _ uintptr) {
		g.drawPanel(newPen(ctx), float64(width), float64(height))
	}
	night.SetDrawFunc(&g.onPanelDraw, 0, &noDestroy)
	appendTo(panel, &night.Widget)
	root.AddOverlay(&panel.Widget)
	panel.Unref()

	if g.cfg.Power {
		power := g.buildPower()
		power.AddCssClass("isekai-power")
		power.SetHalign(gtk.AlignEndValue)
		power.SetValign(gtk.AlignStartValue)
		power.SetMarginTop(g.px(20))
		power.SetMarginEnd(g.px(20))
		root.AddOverlay(&power.Widget)
		power.Unref()
	}

	win.SetChild(&root.Widget)
	root.Unref()
}

// width is the window's: the demo's fixed one, else the monitor's, whose
// share the panel takes as the concept does. 2400 when the display cannot
// say.
func (g *greeter) width() float64 {
	if g.demoW > 0 {
		return g.demoW
	}
	return screenWidth()
}

// screenWidth is the first monitor's logical width. 2400 when the display
// cannot say.
func screenWidth() float64 {
	display := gdk.DisplayGetDefault()
	if display == nil {
		return 2400
	}
	monitors := display.GetMonitors()
	if monitors == nil {
		return 2400
	}
	mon := gdk.MonitorNewFromInternalPtr(monitors.GetItem(0))
	if mon == nil {
		return 2400
	}
	var geo gdk.Rectangle
	mon.GetGeometry(&geo)
	if geo.Width <= 0 {
		return 2400
	}
	return float64(geo.Width)
}

var noDestroy glib.DestroyNotify = func(uintptr) {}

// The concept's canvas, in its own units, and the night panel's share.
const (
	canvasW    = 1600.0
	canvasH    = 1000.0
	panelShare = 0.325
)

// layout settles the panel and the card for a window width: the panel's
// share of it, and the card a little left of the paper's middle. The
// window's width is only known when it is drawn, so the demo window (the
// compositor may have given it another size) lands right too.
func (g *greeter) layout(w float64) {
	if w == g.layoutW {
		return
	}
	g.layoutW = w
	g.panelW = math.Round(panelShare * w)
	if g.panel != nil {
		g.panel.SetSizeRequest(int32(g.panelW), -1)
	}
	if g.card != nil {
		paperW := w - g.panelW
		g.card.SetMarginEnd(int32(math.Round(w - 0.72*paperW)))
	}
}

// drawDecor paints the paper stage: mark, vines, eye-flower, thread and
// the concept's micro text, in the concept's 1600 × 1000 units, then the
// grain.
func (g *greeter) drawDecor(p pen, width, height float64) {
	w, h := width, height
	g.layout(w)

	s := math.Min(w/canvasW, h/canvasH)
	p.Save()
	p.Translate((w-canvasW*s)/2, (h-canvasH*s)/2)
	p.Scale(s, s)

	drawMark(p)
	drawVines(p)
	drawBigEyeFlower(p)
	drawThread(p)
	drawStageText(p)

	p.Restore()

	p.Rect(0, 0, w, h)
	p.Texture(g.grain, 1)
	p.NewPath()
}

// drawPanel paints the night panel: the small eye-flower, its labels,
// the vertical title, the carat and the credit, in 520 × 1000 units.
func (g *greeter) drawPanel(p pen, width, height float64) {
	w, h := width, height

	s := math.Min(w/520, h/1000)
	p.Save()
	p.Translate((w-520*s)/2, (h-1000*s)/2)
	p.Scale(s, s)

	drawSmallEyeFlower(p, 90, 214, 46)
	drawNightText(p, 520)

	p.Restore()

	p.Rect(0, 0, w, h)
	p.Texture(g.panelGrain, 1)
	p.NewPath()
}

// drawMark is the big ヰ.
func drawMark(p pen) {
	p.Text("ヰ", fontPx(fontSerifJP, 580), 610, 120, 0, 0, isInk)
}

// drawVines is the ink scroll with its leaves and rust berries.
func drawVines(p pen) {
	p.Save()
	p.Width(1)
	p.Round()
	p.Color(isInk)

	p.MoveTo(-30, 40)
	p.Curve(160, 70, 230, 190, 178, 300)
	p.Curve(140, 382, 190, 468, 300, 500)
	p.Stroke()
	p.MoveTo(150, -30)
	p.Curve(190, 110, 300, 180, 402, 150)
	p.Curve(470, 130, 516, 170, 540, 240)
	p.Stroke()
	p.MoveTo(-30, 930)
	p.Curve(160, 900, 270, 820, 300, 700)
	p.Curve(318, 630, 290, 580, 250, 545)
	p.Stroke()
	p.MoveTo(40, 1010)
	p.Curve(150, 960, 210, 860, 205, 760)
	p.Stroke()

	for _, l := range []struct{ x, y, rx, ry, rot float64 }{
		{178, 300, 13, 4, -58}, {300, 500, 15, 4, 38}, {402, 150, 14, 4, 16},
		{540, 240, 12, 4, 64}, {300, 700, 15, 4, 114}, {250, 545, 12, 4, -34},
	} {
		p.NewPath()
		p.Ellipse(l.x, l.y, l.rx, l.ry, l.rot)
		p.Color(isInk)
		p.Fill()
	}
	for _, b := range []struct{ x, y, r float64 }{
		{178, 300, 3}, {300, 700, 3.5}, {402, 150, 2.5}, {205, 760, 2.5},
	} {
		p.NewPath()
		p.Circle(b.x, b.y, b.r)
		p.Color(isRust)
		p.Fill()
	}
	p.Restore()
}

// drawThread is the rust thread, from the night panel across the paper.
func drawThread(p pen) {
	p.Save()
	p.Width(1.4)
	p.Round()
	p.Color(isRust)
	p.MoveTo(1310, 800)
	p.Curve(1180, 790, 1120, 730, 1040, 700)
	p.Curve(950, 665, 910, 570, 820, 510)
	p.Curve(754, 452, 710, 560, 640, 620)
	p.Curve(570, 690, 520, 760, 460, 790)
	p.Curve(440, 798, 425, 810, 410, 826)
	p.Stroke()
	for _, b := range []struct{ x, y, r float64 }{{820, 510, 3.4}, {410, 826, 2.4}} {
		p.NewPath()
		p.Circle(b.x, b.y, b.r)
		p.Fill()
	}
	p.Restore()
}

// drawBigEyeFlower is the eye-flower over the mark: ten petals, five of
// them rust, the paper eye with the grey iris.
func drawBigEyeFlower(p pen) { drawEyeFlower(p, 820, 510, 160, true) }

// drawSmallEyeFlower is the panel's small one, in paper lines.
func drawSmallEyeFlower(p pen, cx, cy, r float64) { drawEyeFlower(p, cx, cy, r, false) }

func drawEyeFlower(p pen, cx, cy, r float64, big bool) {
	p.Save()
	p.Translate(cx, cy)
	p.Scale(r/120, r/120)

	if big {
		p.Width(1.2)
		for i := 0; i < 10; i++ {
			rot := float64(i) * 36
			if i == 0 { // petal 1 has a paper fill
				p.NewPath()
				p.Ellipse(0, -72, 22, 54, rot)
				p.Alpha(isPaper, 0.2)
				p.Fill()
			}
			p.NewPath()
			p.Ellipse(0, -72, 22, 54, rot)
			if i%2 == 1 {
				p.Color(isRust)
			} else {
				p.Color(isInk)
			}
			p.Stroke()
		}

		eyePath := func() {
			p.NewPath()
			p.MoveTo(-96, 0)
			p.Curve(-56, -40, 56, -40, 96, 0)
			p.Curve(56, 40, -56, 40, -96, 0)
			p.Close()
		}
		eyePath()
		p.Color(isPaper)
		p.Fill()
		eyePath()
		p.Width(1.4)
		p.Color(isInk)
		p.Stroke()
		p.NewPath()
		p.Circle(0, 0, 34)
		p.Color(isEye)
		p.Fill()
		p.NewPath()
		p.Circle(0, 0, 15)
		p.Color(isInk)
		p.Fill()
		p.NewPath()
		p.Circle(-11, -12, 4.5)
		p.Color(isPaper)
		p.Fill()
		// the little stem
		p.MoveTo(-120, 12)
		p.Curve(-160, 66, -128, 106, -84, 116)
		p.Width(1)
		p.Stroke()
		p.Restore()
		return
	}

	p.Width(1.2)
	p.Color(isPaper)
	for i := 0; i < 8; i++ {
		p.NewPath()
		p.Ellipse(0, -66, 20, 48, float64(i)*45)
		p.Stroke()
	}
	p.NewPath()
	p.MoveTo(-46, 0)
	p.Curve(-30, -27, 30, -27, 46, 0)
	p.Curve(30, 27, -30, 27, -46, 0)
	p.Close()
	p.Stroke()
	p.NewPath()
	p.Circle(0, 0, 12)
	p.Alpha(isPaper, 0.85)
	p.Fill()
	p.Restore()
}

// drawStageText is the concept's micro text: the kicker, the vertical
// phrase, the four tags and the palette.
func drawStageText(p pen) {
	// [CHARACTER STUDY] — 001 / 片目
	kicker := fontPx(fontSans, 9)
	p.TextTracked("CHARACTER STUDY", kicker, 46, 38, 0, 0, 3.06, isInk)
	kw := p.TextWidth("CHARACTER STUDY", kicker, 3.06) + 3.06
	p.TextTracked("— 001 / 片目", kicker, 46+kw, 38, 0, 0, 3.06, fade(isInk, 0.55))

	// 闇と光のあいだで歌う
	p.TextStacked("闇と光のあいだで歌う", fontPx(fontSerifJP, 15), 54, 92, 20, fade(isInk, 0.62))

	// the tags: a hairline, then the words
	tag := func(x, y, line float64, s string) {
		p.Width(1)
		p.Alpha(isInk, 0.55)
		p.Line(x, y, x+line, y)
		p.TextTracked(s, fontPx(fontSans, 8), x+line+9, y, 0, 0.5, 1.92, fade(isInk, 0.6))
	}
	tag(356, 272, 22, "ヰ / A LETTER FROM ELSEWHERE")
	tag(700, 300, 32, "片目 / ONE EYE")
	tag(700, 690, 22, "眼の花 / EYE-FLOWER")
	tag(318, 740, 22, "朱い糸 / RUST THREAD")

	drawPalette(p)
}

// drawPalette is the palette block at the paper's foot: five swatches.
func drawPalette(p pen) {
	swatches := []struct {
		name string
		hex  string
		c    color.NRGBA
	}{
		{"PAPER", "F4F0E8", isPaper},
		{"INK", "14131A", isInk},
		{"RUST", "A6402B", isRust},
		{"EYE", "7F8C94", isEye},
		{"SUN", "F2C230", isSun},
	}
	// PALETTE, written downward
	p.TextStacked("PALETTE", fontPx(fontSans, 7), 56, 884, 9.5, fade(isInk, 0.52))

	x := 74.0
	for _, s := range swatches {
		p.Rect(x, 898, 58, 26)
		p.Color(s.c)
		p.Fill()
		p.Rect(x, 898, 58, 26)
		p.Width(1)
		p.Alpha(isInk, 0.08)
		p.Stroke()
		p.Text(s.name, fontPx(fontSans, 6), x, 932, 0, 0, fade(isInk, 0.56))
		p.Text(s.hex, fontPx(fontSans, 6), x, 942, 0, 0, fade(isInk, 0.56))
		x += 58 + 18
	}
}

// drawNightText is what the night panel says.
func drawNightText(p pen, panelW float64) {
	quiet := fade(isPaper, 0.62)
	// ISEKAI JŌCHO / A VIRTUAL DARK SINGER / KAMITSUBAKI STUDIO
	p.TextTracked("ISEKAI JŌCHO", fontPx(fontSans, 8), 44, 58, 0, 0, 2.4, isPaper)
	p.TextTracked("A VIRTUAL DARK SINGER", fontPx(fontSans, 8), 44, 76, 0, 0, 2.4, quiet)
	p.TextTracked("KAMITSUBAKI STUDIO", fontPx(fontSans, 8), 44, 94, 0, 0, 2.4, quiet)

	// ヰ世界情緒, written downward at the panel's right
	p.TextStacked("ヰ世界情緒", fontPx(fontSerifJP, 84), panelW-48-42, 48, 92, isPaper)

	// 1209 CARAT. FAN CLUB — EST. 12.09
	numW, _ := p.Text("1209", fontPx(fontSerifLat, 136), 38, 1000-138, 0, 1, isPaper)
	ux := 38 + numW + 14
	p.TextTracked("CARAT. FAN CLUB", fontPx(fontSans, 8), ux, 1000-138-20, 0, 0, 2.4, isPaper)
	p.TextTracked("EST. 12.09", fontPx(fontSans, 8), ux, 1000-138, 0, 0, 2.4, fade(isPaper, 0.6))

	// the credit
	p.TextTracked("CHARACTER DESIGN / ORIE", fontPx(fontSans, 7), 44, 1000-44-16, 0, 0, 1.4, quiet)
	p.TextTracked("FAN CONCEPT — NOT AFFILIATED", fontPx(fontSans, 7), 44, 1000-44, 0, 0, 1.4, fade(isPaper, 0.46))
}
