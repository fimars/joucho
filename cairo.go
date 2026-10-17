// Bound from inua (internal/gamesui/cairo.go, Apache-2.0; see NOTICE):
// puregotk binds cairo's types but none of its drawing, so the calls the
// greeter's decorations use are bound here with purego, from the libcairo
// GTK has already loaded. The bitmap type is inua's too.

package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"sync"
	"unsafe"

	"codeberg.org/puregotk/purego"
	"codeberg.org/puregotk/puregotk/v4/cairo"
	"codeberg.org/puregotk/puregotk/v4/pango"
	"codeberg.org/puregotk/puregotk/v4/pangocairo"
)

var cr struct {
	once                              sync.Once
	save, restore, newPath, closePath func(c uintptr)
	fill, stroke, paint               func(c uintptr)
	newSubPath                        func(c uintptr)
	translate, scale, moveTo, lineTo  func(c uintptr, x, y float64)
	rotate, setLineWidth              func(c uintptr, v float64)
	setSourceRGBA                     func(c uintptr, r, g, b, a float64)
	arc, arcNegative                  func(c uintptr, x, y, r, a1, a2 float64)
	rectangle                         func(c uintptr, x, y, w, h float64)
	curveTo                           func(c uintptr, x1, y1, x2, y2, x3, y3 float64)
	setLineCap, setLineJoin           func(c uintptr, v int32)
	imageSurface                      func(format, w, h int32) uintptr
	create                            func(s uintptr) unsafe.Pointer
	destroy, destroySurface           func(p uintptr)
	flushSurface, dirtySurface        func(s uintptr)
	surfaceData                       func(s uintptr) unsafe.Pointer
	surfaceStride                     func(s uintptr) int32
	surfacePattern                    func(s uintptr) uintptr
	setSource                         func(c, p uintptr)
	destroyPattern                    func(p uintptr)
	setExtend, setFilter              func(p uintptr, v int32)
	writePNG                          func(s uintptr, path *byte) int32
}

func loadCairo() {
	cr.once.Do(func() {
		lib, err := purego.Dlopen("libcairo.so.2", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			panic("joucho: " + err.Error())
		}
		for name, f := range map[string]any{
			"cairo_save": &cr.save, "cairo_restore": &cr.restore, "cairo_new_path": &cr.newPath,
			"cairo_close_path": &cr.closePath, "cairo_fill": &cr.fill, "cairo_stroke": &cr.stroke,
			"cairo_paint":     &cr.paint,
			"cairo_translate": &cr.translate, "cairo_scale": &cr.scale,
			"cairo_move_to": &cr.moveTo, "cairo_line_to": &cr.lineTo, "cairo_rotate": &cr.rotate,
			"cairo_new_sub_path": &cr.newSubPath,
			"cairo_set_source":   &cr.setSource, "cairo_pattern_destroy": &cr.destroyPattern,
			"cairo_set_line_width": &cr.setLineWidth, "cairo_set_source_rgba": &cr.setSourceRGBA,
			"cairo_arc": &cr.arc, "cairo_arc_negative": &cr.arcNegative, "cairo_rectangle": &cr.rectangle,
			"cairo_curve_to": &cr.curveTo, "cairo_set_line_cap": &cr.setLineCap,
			"cairo_set_line_join":        &cr.setLineJoin,
			"cairo_image_surface_create": &cr.imageSurface,
			"cairo_create":               &cr.create, "cairo_destroy": &cr.destroy, "cairo_surface_destroy": &cr.destroySurface,
			"cairo_surface_flush": &cr.flushSurface, "cairo_surface_mark_dirty": &cr.dirtySurface,
			"cairo_image_surface_get_data": &cr.surfaceData, "cairo_image_surface_get_stride": &cr.surfaceStride,
			"cairo_pattern_create_for_surface": &cr.surfacePattern, "cairo_pattern_set_extend": &cr.setExtend,
			"cairo_pattern_set_filter":   &cr.setFilter,
			"cairo_surface_write_to_png": &cr.writePNG,
		} {
			purego.RegisterLibFunc(f, lib, name)
		}
	})
}

// pen draws on one frame's cairo context, in the decorations' own units.
type pen struct {
	c   uintptr
	ctx *cairo.Context
}

func newPen(ctx *cairo.Context) pen {
	loadCairo()
	return pen{ctx.GoPointer(), ctx}
}

func (p pen) Save()                                { cr.save(p.c) }
func (p pen) Restore()                             { cr.restore(p.c) }
func (p pen) Translate(x, y float64)               { cr.translate(p.c, x, y) }
func (p pen) Scale(x, y float64)                   { cr.scale(p.c, x, y) }
func (p pen) Rotate(a float64)                     { cr.rotate(p.c, a) }
func (p pen) MoveTo(x, y float64)                  { cr.moveTo(p.c, x, y) }
func (p pen) LineTo(x, y float64)                  { cr.lineTo(p.c, x, y) }
func (p pen) Close()                               { cr.closePath(p.c) }
func (p pen) NewPath()                             { cr.newPath(p.c) }
func (p pen) Fill()                                { cr.fill(p.c) }
func (p pen) Stroke()                              { cr.stroke(p.c) }
func (p pen) Width(w float64)                      { cr.setLineWidth(p.c, w) }
func (p pen) Rect(x, y, w, h float64)              { cr.rectangle(p.c, x, y, w, h) }
func (p pen) Color(c color.NRGBA)                  { p.Alpha(c, 1) }
func (p pen) Round()                               { cr.setLineCap(p.c, 1); cr.setLineJoin(p.c, 1) }
func (p pen) Curve(x1, y1, x2, y2, x3, y3 float64) { cr.curveTo(p.c, x1, y1, x2, y2, x3, y3) }

// Alpha sets a colour at a fraction of its opacity.
func (p pen) Alpha(c color.NRGBA, a float64) {
	cr.setSourceRGBA(p.c, float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, float64(c.A)/255*a)
}

// Arc adds an arc to the path, as a new piece of it.
func (p pen) Arc(x, y, r, a1, a2 float64) {
	cr.newSubPath(p.c)
	cr.arc(p.c, x, y, r, a1, a2)
}

// ArcOn continues the path along an arc, clockwise; ArcBack the other way.
func (p pen) ArcOn(x, y, r, a1, a2 float64)   { cr.arc(p.c, x, y, r, a1, a2) }
func (p pen) ArcBack(x, y, r, a1, a2 float64) { cr.arcNegative(p.c, x, y, r, a1, a2) }

// Circle adds a circle to the path.
func (p pen) Circle(x, y, r float64) { p.Arc(x, y, r, 0, 2*math.Pi) }

// Ellipse adds an ellipse turned by rot to the path (for leaves).
func (p pen) Ellipse(x, y, rx, ry, rot float64) {
	p.Save()
	p.Translate(x, y)
	p.Rotate(rot * math.Pi / 180)
	p.Scale(rx, ry)
	p.Arc(0, 0, 1, 0, 2*math.Pi)
	p.Restore()
}

// Line strokes one segment.
func (p pen) Line(x1, y1, x2, y2 float64) {
	p.MoveTo(x1, y1)
	p.LineTo(x2, y2)
	p.Stroke()
}

// Text sets s in font (a Pango description, size in points) with its
// anchor at x, y, placed by ax and ay: 0 left or top, 0.5 centre, 1 right
// or bottom; c's alpha fades it. It returns the text's width and height.
func (p pen) Text(s, font string, x, y, ax, ay float64, c color.NRGBA) (float64, float64) {
	l := pangocairo.CreateLayout(p.ctx)
	defer l.Unref()
	d := pango.FontDescriptionFromString(font)
	l.SetFontDescription(d)
	d.Free()
	l.SetText(s, -1)
	var w, h int32
	l.GetPixelSize(&w, &h)
	p.Color(c)
	p.MoveTo(x-float64(w)*ax, y-float64(h)*ay)
	pangocairo.ShowLayout(p.ctx, l)
	p.NewPath()
	return float64(w), float64(h)
}

// TextTracked is Text with tracking (letter-spacing) between characters,
// for the concept's micro labels.
func (p pen) TextTracked(s, font string, x, y, ax, ay, track float64, c color.NRGBA) (float64, float64) {
	runes := []rune(s)
	var widths []float64
	var w, h float64
	l := pangocairo.CreateLayout(p.ctx)
	d := pango.FontDescriptionFromString(font)
	l.SetFontDescription(d)
	d.Free()
	for _, r := range runes {
		l.SetText(string(r), -1)
		var cw, ch int32
		l.GetPixelSize(&cw, &ch)
		widths = append(widths, float64(cw))
		w += float64(cw)
		h = math.Max(h, float64(ch))
	}
	l.Unref()
	if n := len(runes); n > 1 {
		w += track * float64(n-1)
	}
	start := x - w*ax
	for i, r := range runes {
		p.Text(string(r), font, start, y, 0, ay, c)
		start += widths[i] + track
	}
	return w, h
}

// TextStacked writes s downward, one character per line, as the concept's
// vertical writing does; x is the line's centre, y the first character's
// top. It returns the block's width and height.
func (p pen) TextStacked(s, font string, x, y, advance float64, c color.NRGBA) (float64, float64) {
	runes := []rune(s)
	var w float64
	for i, r := range runes {
		cw, _ := p.Text(string(r), font, x, y+float64(i)*advance, 0.5, 0, c)
		w = math.Max(w, cw)
	}
	return w, float64(len(runes)) * advance
}

// TextWidth measures s in font with tracking, without drawing it.
func (p pen) TextWidth(s, font string, track float64) float64 {
	runes := []rune(s)
	var w float64
	l := pangocairo.CreateLayout(p.ctx)
	defer l.Unref()
	d := pango.FontDescriptionFromString(font)
	l.SetFontDescription(d)
	d.Free()
	for _, r := range runes {
		l.SetText(string(r), -1)
		var cw, ch int32
		l.GetPixelSize(&cw, &ch)
		w += float64(cw)
	}
	if n := len(runes); n > 1 {
		w += track * float64(n-1)
	}
	return w
}

// bitmap is a picture drawn once off screen and painted from then on.
type bitmap struct {
	s    uintptr
	w, h int
}

// newBitmap makes a transparent image of w by h pixels; Free frees it.
func newBitmap(w, h int) *bitmap {
	loadCairo()
	return &bitmap{cr.imageSurface(0, int32(w), int32(h)), w, h} // CAIRO_FORMAT_ARGB32
}

func (b *bitmap) Free() { cr.destroySurface(b.s) }

// Speckle scatters n dots of c over the bitmap, each at a random fraction
// of alpha a: paper's grain.
func (b *bitmap) Speckle(r *rand.Rand, n int, c color.NRGBA, a float64) {
	cr.flushSurface(b.s)
	stride := int(cr.surfaceStride(b.s))
	px := unsafe.Slice((*byte)(cr.surfaceData(b.s)), stride*b.h)
	for range n {
		x, y := r.IntN(b.w), r.IntN(b.h)
		al := a * (0.3 + 0.7*r.Float64())
		i := y*stride + x*4 // premultiplied BGRA, little-endian ARGB32
		over := func(dst *byte, v uint8) { *dst = uint8(float64(v)*al + float64(*dst)*(1-al)) }
		over(&px[i], c.B)
		over(&px[i+1], c.G)
		over(&px[i+2], c.R)
		px[i+3] = uint8(al*255 + float64(px[i+3])*(1-al))
	}
	cr.dirtySurface(b.s)
}

// Texture fills the current path with im repeated, k units to its pixel.
func (p pen) Texture(im *bitmap, k float64) {
	p.Save()
	p.Scale(k, k)
	pat := cr.surfacePattern(im.s)
	cr.setExtend(pat, 1) // CAIRO_EXTEND_REPEAT
	cr.setSource(p.c, pat)
	cr.destroyPattern(pat)
	p.Fill()
	p.Restore()
}

// SavePNG writes the bitmap to path as a PNG.
func (b *bitmap) SavePNG(path string) error {
	cr.flushSurface(b.s)
	cs := append([]byte(path), 0)
	if res := cr.writePNG(b.s, &cs[0]); res != 0 {
		return fmt.Errorf("cairo_surface_write_to_png: %d", res)
	}
	return nil
}
