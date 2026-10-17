package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"codeberg.org/puregotk/puregotk/v4/gdk"
)

// The built-in themes are inua's (ui/themes/), Apache-2.0, so the greeter
// is dressed in the same colours as the shell. See NOTICE.
//
//go:embed assets/themes/*.css
var assets embed.FS

//go:embed assets/style.css
var greeterStyle string

// Themes are the built-in colour themes, inua's names.
var Themes = []string{"aojashin", "bunko", "denko", "isekai", "rinko", "suimen", "washi", "yoru"}

// themeCSS is a theme's colour tokens: inua's built-in theme of that name
// (the default theme for a name that is not one), then themes/NAME.css
// from the config directory over it, so a theme of your own needs only
// the tokens it changes.
func (g *greeter) themeCSS(name string) string {
	base := name
	if !slices.Contains(Themes, base) {
		base = defaultTheme
	}
	data, _ := assets.ReadFile("assets/themes/" + base + ".css")
	css := string(data)
	if name != "" {
		custom := filepath.Join(filepath.Dir(g.cfgPath), "themes", name+".css")
		if b, err := os.ReadFile(custom); err == nil {
			css += "\n" + string(b)
		} else if !os.IsNotExist(err) {
			logf("theme: %v", err)
		}
	}
	return css
}

// builtinFont is the window font stack: the shell's own (ui/style.css),
// then CJK, with the system's sans as the last resort.
var builtinFont = []string{"Zen Kaku Gothic New", "Noto Sans CJK JP", "Noto Sans CJK SC"}

// fontCSS overrides the window font when the settings name one. The
// isekai dress gives the card a face of its own (style.css), so the
// setting is repeated for those selectors and stays the user's choice.
func (g *greeter) fontCSS() string {
	if g.cfg.Font == "" {
		return ""
	}
	sel := "window.greet"
	if g.isekai() {
		sel += ", window.greet.isekai, window.greet.isekai .greet-time, window.greet.isekai .greet-date, window.greet.isekai button.primary"
	}
	return fmt.Sprintf("%s { font-family: %s; }\n", sel, fontStack(g.cfg.Font))
}

// fontStack is one family first, then the built-in stack.
func fontStack(font string) string {
	parts := make([]string, 0, len(builtinFont)+2)
	parts = append(parts, strconv.Quote(font))
	for _, f := range builtinFont {
		parts = append(parts, strconv.Quote(f))
	}
	parts = append(parts, "sans-serif")
	return strings.Join(parts, ", ")
}

// warnMissingFont says so in greetd's log when the font named in the
// settings is not installed for the greeter user: fonts in your own home
// are invisible to it, and a silent fallback is hard to explain. The
// built-in stack is the shell's and needs no such warning.
func (g *greeter) warnMissingFont() {
	family := g.cfg.Font
	if family == "" {
		return
	}
	fc, err := exec.LookPath("fc-match")
	if err != nil {
		return
	}
	out, err := exec.Command(fc, "-f", "%{family}", family).Output()
	if err != nil {
		return
	}
	if got := strings.TrimSpace(string(out)); got != "" && !strings.EqualFold(got, family) {
		logf("font: %q is not installed for the greeter user (fontconfig picks %q); install it system-wide, or set another font in %s", family, got, g.cfgPath)
	}
}

var (
	cssPx      = regexp.MustCompile(`[0-9]*\.?[0-9]+px`)
	fontSizePx = regexp.MustCompile(`(font-size\s*:\s*)([0-9]*\.?[0-9]+)px`)
)

// scaleUI multiplies every pixel size of a stylesheet by percent/100
// (ui_scale).
func scaleUI(css string, percent int) string {
	if percent == 100 {
		return css
	}
	return cssPx.ReplaceAllStringFunc(css, func(m string) string {
		px, err := strconv.ParseFloat(strings.TrimSuffix(m, "px"), 64)
		if err != nil {
			return m
		}
		return scaledPx(px, percent)
	})
}

// scaleText multiplies the pixel font sizes of a stylesheet by
// percent/100 (text_scale), leaving every other size as it is. It is
// inua's shell.ScaleText, so the setting means the same thing here.
func scaleText(css string, percent int) string {
	if percent == 100 {
		return css
	}
	return fontSizePx.ReplaceAllStringFunc(css, func(m string) string {
		sub := fontSizePx.FindStringSubmatch(m)
		px, err := strconv.ParseFloat(sub[2], 64)
		if err != nil {
			return m
		}
		return sub[1] + scaledPx(px, percent)
	})
}

func scaledPx(px float64, percent int) string {
	px = float64(int(px*float64(percent)+0.5)) / 100 // to 0.01px
	return strconv.FormatFloat(px, 'f', -1, 64) + "px"
}

// automaticUIScale compares the first monitor's logical DPI with the
// design's 96 DPI, then picks a predictable scale for the greeter.
func automaticUIScale() int {
	var dpi float64
	if display := gdk.DisplayGetDefault(); display != nil {
		if monitors := display.GetMonitors(); monitors != nil {
			if mon := gdk.MonitorNewFromInternalPtr(monitors.GetItem(0)); mon != nil {
				var geo gdk.Rectangle
				mon.GetGeometry(&geo)
				if widthMm := mon.GetWidthMm(); geo.Width > 0 && widthMm > 0 {
					dpi = float64(geo.Width) * 25.4 / float64(widthMm)
				}
			}
		}
	}
	if dpi <= 0 {
		logf("ui_scale: automatic 100%% (monitor logical DPI unavailable)")
		return 100
	}
	scale := scaleForDPI(dpi)
	logf("ui_scale: automatic %d%% (%.1f logical DPI)", scale, dpi)
	return scale
}

func scaleForDPI(dpi float64) int {
	if dpi <= 0 {
		return 100
	}
	scale := int(math.Round(dpi/96*100/20)) * 20
	if scale < 100 {
		return 100
	}
	if scale > maxUIScale {
		return maxUIScale
	}
	return scale
}

var defineColor = regexp.MustCompile(`@define-color\s+([\w-]+)\s+(#[0-9a-fA-F]{6})\s*;`)

// themeColors reads a theme's tokens: name → "#rrggbb".
func themeColors(css string) map[string]string {
	colors := map[string]string{}
	for _, m := range defineColor.FindAllStringSubmatch(css, -1) {
		colors[m[1]] = m[2]
	}
	return colors
}

// groundCSS styles the window's ground: the picture from the settings, or
// the theme's field texture, as the shell's wallpaper windows show it.
func (g *greeter) groundCSS() string {
	colors := themeColors(g.themeCSS(g.cfg.Theme))
	ground := colors["app"]
	if ground == "" {
		ground = "#F2EDE2"
	}
	if g.cfg.Wallpaper != "" {
		if _, err := os.Stat(g.cfg.Wallpaper); err == nil {
			return fmt.Sprintf(`window.greet {
  background-color: %s;
  background-image: url("%s");
  background-repeat: no-repeat;
  background-size: cover;
  background-position: center;
}
`, ground, fileURL(g.cfg.Wallpaper))
		}
		logf("wallpaper: %s: no such file; using the theme's texture", g.cfg.Wallpaper)
	}
	field := colors["field"]
	if field == "" {
		field = "#BDB7AC"
	}
	tile, err := writeFieldTile(field)
	if err != nil {
		logf("background: %v", err)
		return fmt.Sprintf("window.greet { background-color: %s; }\n", ground)
	}
	return fmt.Sprintf(`window.greet {
  background-color: %s;
  background-image: url("%s");
  background-repeat: repeat;
  background-size: %dpx %dpx;
  background-position: 0 0;
}
`, ground, fileURL(tile), tileW, tileH)
}

// The field texture: a run of 3 × 1.5 px every 6 px, rows every 26 px
// (CSS px), as inua's shell draws it.
const (
	tileW, tileH = 6, 26
	tileScale    = 4
)

// writeFieldTile draws one tile of the texture as a PNG and returns its
// path. The greeter runs as its own user, so it goes to the temporary
// directory.
func writeFieldTile(field string) (string, error) {
	c, err := parseHex(field)
	if err != nil {
		return "", err
	}
	alpha := 0.55 * 255
	c.A = uint8(alpha + 0.5)
	img := image.NewNRGBA(image.Rect(0, 0, tileW*tileScale, tileH*tileScale))
	for y := 0; y < tileScale*3/2; y++ { // 1.5 px
		for x := 0; x < 3*tileScale; x++ { // 3 px
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "joucho-field-*.png")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(buf.Bytes()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func parseHex(s string) (color.NRGBA, error) {
	if len(s) != 7 || s[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("colour %q: want #rrggbb", s)
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("colour %q: %w", s, err)
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, nil
}

func fileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}
