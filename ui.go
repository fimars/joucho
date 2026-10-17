// joucho is the login screen for inua, in inua's look: a Wayland
// greeter for greetd, written in Go and GTK4 with the same toolkit as the
// shell (puregotk) and the same colour themes. greetd authenticates (PAM)
// and starts the session; this program only asks.
package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/puregotk/puregotk/v4/gdk"
	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/glib"
	"codeberg.org/puregotk/puregotk/v4/gtk"
	"github.com/godbus/dbus/v5"
)

// keyEscape is X11's Escape keysym, as GTK reports it.
const keyEscape = 0xff1b

// greeter is the whole program: the window, its state and its handlers.
type greeter struct {
	app          *gtk.Application
	cfg          Config
	cfgPath      string
	demo         bool
	i18n         *i18n
	sessionsDirs []string

	users    []User
	sessions []Session
	user     User
	session  Session

	client   *greetdClient // the greetd conversation under way
	awaiting bool          // greetd waits for what the user types
	finished bool          // the session is starting; the greeter is done

	css *gtk.CssProvider // kept for the lifetime of the window

	// The isekai dress (decor.go): paper grain, over the stage and over
	// the night panel, and its two drawing callbacks.
	demoW       float64 // the demo window's width, 0 when fullscreen
	grain       *bitmap
	panelGrain  *bitmap
	panelW      float64 // the night panel's width, in logical pixels
	layoutW     float64 // the width the isekai layout was computed for
	panel       *gtk.Box
	card        *gtk.Box
	onDecorDraw gtk.DrawingAreaDrawFunc
	onPanelDraw gtk.DrawingAreaDrawFunc

	win         *gtk.ApplicationWindow
	timeLbl     *gtk.Label
	dateLbl     *gtk.Label
	userBtn     *gtk.Button
	userName    *gtk.Label
	pass        *gtk.PasswordEntry
	sessionBtn  *gtk.Button
	sessionName *gtk.Label
	status      *gtk.Label
	signIn      *gtk.Button
	userPop     *gtk.Popover
	sessionPop  *gtk.Popover
	userChecks  []*gtk.Image
	sessChecks  []*gtk.Image

	// Handlers live as long as the program: puregotk makes one native
	// trampoline per Go function pointer, so widgets share them and tell
	// each other apart by name, as inua's UI does.
	onActivate     func(gio.Application)
	onSignIn       func(gtk.Button)
	onPassActivate func(gtk.PasswordEntry)
	onKey          func(gtk.EventControllerKey, uint32, uint32, gdk.ModifierType) bool
	onRow          func(gtk.Button)
	onPower        func(gtk.Button)
	onTick         glib.SourceFunc
	onQuit         glib.SourceFunc
}

// --- main loop posting ---

var (
	queueMu sync.Mutex
	queue   []func()
	drain   glib.SourceFunc = func(uintptr) bool {
		queueMu.Lock()
		q := queue
		queue = nil
		queueMu.Unlock()
		for _, f := range q {
			f()
		}
		return false
	}
)

// post runs f on the GTK main loop thread; the auth and D-Bus goroutines
// hand their results over with it.
func post(f func()) {
	queueMu.Lock()
	first := len(queue) == 0
	queue = append(queue, f)
	queueMu.Unlock()
	if first {
		glib.IdleAdd(&drain, 0)
	}
}

// appendTo adds a widget to box and drops Go's reference, leaving box as
// the only owner (inua's gtkx.Append).
func appendTo(box *gtk.Box, w *gtk.Widget) {
	box.Append(w)
	w.Unref()
}

// --- start ---

// activate builds the window and starts the first conversation.
func (g *greeter) activate() {
	g.onSignIn = func(gtk.Button) { g.submit() }
	g.onPassActivate = func(gtk.PasswordEntry) { g.submit() }
	g.onRow = func(b gtk.Button) { g.rowClicked(b) }
	g.onPower = func(b gtk.Button) { g.powerClicked(b) }
	g.onKey = func(_ gtk.EventControllerKey, keyval, _ uint32, _ gdk.ModifierType) bool {
		if keyval == keyEscape {
			g.pass.SetText("")
			return true
		}
		return false
	}

	g.loadStyle()
	g.win = g.build()
	g.win.Present()
	g.tick()
	g.onTick = func(uintptr) bool { g.tick(); return true }
	glib.TimeoutAdd(1000, &g.onTick, 0)
	g.startAuth()
}

// loadStyle builds the stylesheet: the theme's tokens, inua's look scaled
// by text_scale, and the ground.
func (g *greeter) loadStyle() {
	g.warnMissingFont()
	css := g.themeCSS(g.cfg.Theme) + "\n" + scaleText(greeterStyle, g.cfg.TextScale) + "\n" + g.fontCSS() + "\n" + g.groundCSS()
	g.css = gtk.NewCssProvider()
	g.css.LoadFromString(css)
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), g.css, uint32(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION))
}

// tick keeps the clock and the date right.
func (g *greeter) tick() {
	now := time.Now()
	g.timeLbl.SetText(g.i18n.clock(now, g.cfg.ClockSeconds))
	g.dateLbl.SetText(g.i18n.date(now))
}

// --- window and widgets ---

// build makes the window: theme "isekai" dresses it as the concept, any
// other theme as the plain greeter.
func (g *greeter) build() *gtk.ApplicationWindow {
	win := gtk.NewApplicationWindow(g.app)
	win.AddCssClass("greet")
	win.SetDecorated(false)
	win.SetTitle("joucho")
	if g.demo {
		if g.isekai() {
			// The concept's canvas at one unit to the pixel.
			win.SetDefaultSize(1600, 1000)
			g.demoW = 1600
		} else {
			win.SetDefaultSize(430, 600)
		}
	} else {
		win.Fullscreen()
	}

	if g.isekai() {
		g.buildIsekai(win)
	} else {
		g.buildPlain(win)
	}

	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(&g.onKey)
	win.AddController(&keys.EventController)
	return win
}

// buildPlain is the greeter without dress: the card on the theme's ground,
// the power buttons in the corner.
func (g *greeter) buildPlain(win *gtk.ApplicationWindow) {
	overlay := gtk.NewOverlay()

	root := gtk.NewBox(gtk.OrientationVerticalValue, 0)
	top := gtk.NewBox(gtk.OrientationVerticalValue, 0)
	top.SetVexpand(true)
	appendTo(root, &top.Widget)
	card := g.buildCard()
	card.SetHalign(gtk.AlignCenterValue)
	appendTo(root, &card.Widget)
	bottom := gtk.NewBox(gtk.OrientationVerticalValue, 0)
	bottom.SetVexpand(true)
	appendTo(root, &bottom.Widget)
	overlay.SetChild(&root.Widget)
	root.Unref()

	if g.cfg.Power {
		power := g.buildPower()
		power.SetHalign(gtk.AlignEndValue)
		power.SetValign(gtk.AlignStartValue)
		power.SetMarginTop(16)
		power.SetMarginEnd(16)
		overlay.AddOverlay(&power.Widget)
		power.Unref()
	}

	win.SetChild(&overlay.Widget)
	overlay.Unref()
}

// buildCard is the sign-in card.
func (g *greeter) buildCard() *gtk.Box {
	card := gtk.NewBox(gtk.OrientationVerticalValue, 12)
	card.AddCssClass("greet-card")
	card.SetSizeRequest(400, -1)

	head := gtk.NewBox(gtk.OrientationVerticalValue, 2)
	g.timeLbl = gtk.NewLabel("")
	g.timeLbl.AddCssClass("greet-time")
	g.timeLbl.SetHalign(gtk.AlignStartValue)
	appendTo(head, &g.timeLbl.Widget)
	g.dateLbl = gtk.NewLabel("")
	g.dateLbl.AddCssClass("greet-date")
	g.dateLbl.SetHalign(gtk.AlignStartValue)
	appendTo(head, &g.dateLbl.Widget)
	appendTo(card, &head.Widget)

	sep := gtk.NewSeparator(gtk.OrientationHorizontalValue)
	appendTo(card, &sep.Widget)

	// The user.
	g.userBtn = gtk.NewButton()
	g.userBtn.AddCssClass("row")
	g.userBtn.SetName("user")
	g.userBtn.SetTooltipText(g.i18n.T("User"))
	userRow := gtk.NewBox(gtk.OrientationHorizontalValue, 8)
	userIcon := gtk.NewImageFromIconName("avatar-default-symbolic")
	userIcon.SetPixelSize(16)
	appendTo(userRow, &userIcon.Widget)
	g.userName = gtk.NewLabel(g.user.display())
	g.userName.AddCssClass("name")
	g.userName.SetXalign(0)
	g.userName.SetHexpand(true)
	g.userName.SetEllipsize(3) // pango.EllipsizeEnd
	appendTo(userRow, &g.userName.Widget)
	userArrow := gtk.NewImageFromIconName("pan-down-symbolic")
	userArrow.SetPixelSize(16)
	userArrow.AddCssClass("dim-icon")
	appendTo(userRow, &userArrow.Widget)
	g.userBtn.SetChild(&userRow.Widget)
	userRow.Unref()
	g.userBtn.ConnectClicked(&g.onRow)
	appendTo(card, &g.userBtn.Widget)
	g.buildUserMenu()

	// The password.
	g.pass = gtk.NewPasswordEntry()
	g.pass.SetPropertyPlaceholderText(g.i18n.T("Password"))
	g.pass.SetSensitive(false)
	g.pass.ConnectActivate(&g.onPassActivate)
	appendTo(card, &g.pass.Widget)

	// The session.
	g.sessionBtn = gtk.NewButton()
	g.sessionBtn.AddCssClass("row")
	g.sessionBtn.SetName("session")
	g.sessionBtn.SetTooltipText(g.i18n.T("Session"))
	sessRow := gtk.NewBox(gtk.OrientationHorizontalValue, 8)
	sessIcon := gtk.NewImageFromIconName(g.sessionIcon(g.session))
	sessIcon.SetPixelSize(16)
	appendTo(sessRow, &sessIcon.Widget)
	g.sessionName = gtk.NewLabel(g.session.Name)
	g.sessionName.AddCssClass("name")
	g.sessionName.SetXalign(0)
	g.sessionName.SetHexpand(true)
	g.sessionName.SetEllipsize(3)
	appendTo(sessRow, &g.sessionName.Widget)
	sessArrow := gtk.NewImageFromIconName("pan-down-symbolic")
	sessArrow.SetPixelSize(16)
	sessArrow.AddCssClass("dim-icon")
	appendTo(sessRow, &sessArrow.Widget)
	g.sessionBtn.SetChild(&sessRow.Widget)
	sessRow.Unref()
	g.sessionBtn.ConnectClicked(&g.onRow)
	appendTo(card, &g.sessionBtn.Widget)
	g.buildSessionMenu()

	// What greetd says: an error, or a message of its own.
	g.status = gtk.NewLabel("")
	g.status.AddCssClass("status")
	g.status.SetXalign(0)
	g.status.SetWrap(true)
	g.status.SetMaxWidthChars(44)
	g.status.SetVisible(false)
	appendTo(card, &g.status.Widget)

	g.signIn = gtk.NewButton()
	g.signIn.AddCssClass("primary")
	signInLabel := gtk.NewLabel(g.i18n.T("Sign in"))
	g.signIn.SetChild(&signInLabel.Widget)
	signInLabel.Unref()
	g.signIn.SetSensitive(false)
	g.signIn.ConnectClicked(&g.onSignIn)
	appendTo(card, &g.signIn.Widget)

	return card
}

// sessionIcon is a session's own icon, else a display.
func (g *greeter) sessionIcon(s Session) string {
	if s.Icon != "" {
		return s.Icon
	}
	return "video-display-symbolic"
}

// buildUserMenu makes the list of users the user button opens.
func (g *greeter) buildUserMenu() {
	g.userPop = gtk.NewPopover()
	g.userPop.AddCssClass("greet-menu")
	g.userPop.SetHasArrow(false)
	g.userPop.SetParent(&g.userBtn.Widget)
	box := gtk.NewBox(gtk.OrientationVerticalValue, 2)
	for i := range g.users {
		u := g.users[i]
		b := gtk.NewButton()
		b.AddCssClass("menu-item")
		b.SetName(fmt.Sprintf("user:%d", i))
		b.SetTooltipText(u.Name)
		row := gtk.NewBox(gtk.OrientationHorizontalValue, 8)
		icon := gtk.NewImageFromIconName("avatar-default-symbolic")
		icon.SetPixelSize(16)
		appendTo(row, &icon.Widget)
		label := gtk.NewLabel(u.display())
		label.SetXalign(0)
		label.SetHexpand(true)
		appendTo(row, &label.Widget)
		check := gtk.NewImageFromIconName("object-select-symbolic")
		check.SetPixelSize(16)
		check.AddCssClass("check")
		check.SetVisible(u.Name == g.user.Name)
		g.userChecks = append(g.userChecks, check)
		appendTo(row, &check.Widget)
		b.SetChild(&row.Widget)
		row.Unref()
		b.ConnectClicked(&g.onRow)
		appendTo(box, &b.Widget)
	}
	g.userPop.SetChild(&box.Widget)
	box.Unref()
}

// buildSessionMenu makes the list of sessions the session button opens.
func (g *greeter) buildSessionMenu() {
	g.sessionPop = gtk.NewPopover()
	g.sessionPop.AddCssClass("greet-menu")
	g.sessionPop.SetHasArrow(false)
	g.sessionPop.SetParent(&g.sessionBtn.Widget)
	box := gtk.NewBox(gtk.OrientationVerticalValue, 2)
	for i := range g.sessions {
		s := g.sessions[i]
		b := gtk.NewButton()
		b.AddCssClass("menu-item")
		b.SetName(fmt.Sprintf("session:%d", i))
		b.SetTooltipText(s.File)
		row := gtk.NewBox(gtk.OrientationHorizontalValue, 8)
		icon := gtk.NewImageFromIconName(g.sessionIcon(s))
		icon.SetPixelSize(16)
		appendTo(row, &icon.Widget)
		label := gtk.NewLabel(s.Name)
		label.SetXalign(0)
		label.SetHexpand(true)
		appendTo(row, &label.Widget)
		check := gtk.NewImageFromIconName("object-select-symbolic")
		check.SetPixelSize(16)
		check.AddCssClass("check")
		check.SetVisible(s.File == g.session.File)
		g.sessChecks = append(g.sessChecks, check)
		appendTo(row, &check.Widget)
		b.SetChild(&row.Widget)
		row.Unref()
		b.ConnectClicked(&g.onRow)
		appendTo(box, &b.Widget)
	}
	g.sessionPop.SetChild(&box.Widget)
	box.Unref()
}

// powerActions are the buttons in the corner.
var powerActions = []struct{ name, icon, tip string }{
	{"Suspend", "system-suspend-symbolic", "Suspend"},
	{"Reboot", "system-reboot-symbolic", "Restart"},
	{"PowerOff", "system-shutdown-symbolic", "Shut down"},
}

// buildPower makes the suspend, restart and power off buttons.
func (g *greeter) buildPower() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontalValue, 4)
	box.AddCssClass("greet-power")
	for _, a := range powerActions {
		b := gtk.NewButtonFromIconName(a.icon)
		b.AddCssClass("plain")
		b.SetName(a.name)
		b.SetTooltipText(g.i18n.T(a.tip))
		b.ConnectClicked(&g.onPower)
		appendTo(box, &b.Widget)
	}
	return box
}

// --- choosing ---

// rowClicked tells the user and session buttons and their menu items
// apart by name: a plain "user" or "session" is the button that opens
// the menu, an "user:3" or "session:1" is an item in it.
func (g *greeter) rowClicked(b gtk.Button) {
	kind, idx, _ := strings.Cut(b.GetName(), ":")
	if idx == "" {
		switch kind {
		case "user":
			g.userPop.Popup()
		case "session":
			g.sessionPop.Popup()
		}
		return
	}
	i, _ := strconv.Atoi(idx)
	switch {
	case kind == "user" && i >= 0 && i < len(g.users):
		g.pickUser(i)
		g.userPop.Popdown()
	case kind == "session" && i >= 0 && i < len(g.sessions):
		g.pickSession(i)
		g.sessionPop.Popdown()
	}
}

func (g *greeter) pickUser(i int) {
	u := g.users[i]
	if u.Name == g.user.Name {
		return
	}
	g.user = u
	g.userName.SetText(u.display())
	for j, c := range g.userChecks {
		c.SetVisible(g.users[j].Name == u.Name)
	}
	g.pass.SetText("")
	g.startAuth()
}

func (g *greeter) pickSession(i int) {
	s := g.sessions[i]
	if s.File == g.session.File {
		return
	}
	g.session = s
	g.sessionName.SetText(s.Name)
	for j, c := range g.sessChecks {
		c.SetVisible(g.sessions[j].File == s.File)
	}
}

// --- authentication ---

// startAuth starts a greetd conversation for the chosen user. An old one
// is cancelled first; the password prompt follows from greetd's answer.
func (g *greeter) startAuth() {
	g.setAwaiting(false)
	g.setStatus("", false)
	if g.demo {
		g.setAwaiting(true)
		return
	}
	if g.client != nil {
		g.client.cancel()
		g.client.close()
		g.client = nil
	}
	c, err := greetdDial()
	if err != nil {
		g.setStatus(err.Error(), true)
		g.signIn.SetSensitive(true) // to try again
		return
	}
	if err := c.createSession(g.user.Name); err != nil {
		c.close()
		g.setStatus(err.Error(), true)
		return
	}
	g.client = c
	go g.pump(c)
}

// pump reads greetd's answers and hands them to the main loop.
func (g *greeter) pump(c *greetdClient) {
	for {
		resp, err := c.read()
		if err != nil {
			post(func() { g.authClosed(c, err) })
			return
		}
		post(func() { g.authMessage(c, resp) })
		if resp.Type == "success" {
			return
		}
	}
}

// authClosed runs when a conversation ends without a session.
func (g *greeter) authClosed(c *greetdClient, err error) {
	if g.client != c {
		return // a conversation since cancelled
	}
	g.client = nil
	c.close()
	if g.finished {
		return
	}
	logf("greetd: %v", err)
	g.setStatus(g.i18n.T("Lost the connection to greetd."), true)
}

// authMessage follows one answer from greetd.
func (g *greeter) authMessage(c *greetdClient, r greetdResponse) {
	if g.client != c {
		return
	}
	switch r.Type {
	case "auth_message":
		switch r.AuthMessageType {
		case "secret", "visible":
			// PAM's standard prompt needs no line of its own.
			if msg := strings.TrimSpace(r.AuthMessage); msg != "" && !isPasswordPrompt(msg) {
				g.setStatus(r.AuthMessage, false)
			}
			g.setAwaiting(true)
		case "info":
			g.setStatus(r.AuthMessage, false)
			go func() { _ = c.postEmpty() }() // let PAM go on
		case "error":
			g.setStatus(r.AuthMessage, true)
			go func() { _ = c.postEmpty() }()
		}
	case "success":
		g.finished = true
		if err := c.startSession(g.session.Command()); err != nil {
			g.finished = false
			g.client = nil
			c.close()
			g.setStatus(err.Error(), true)
			g.signIn.SetSensitive(true)
			return
		}
		logf("starting %s for %s", g.session.File, g.user.Name)
		// greetd starts the session; the greeter goes away.
		g.onQuit = func(uintptr) bool { g.app.Quit(); return false }
		glib.TimeoutAdd(150, &g.onQuit, 0)
	case "error":
		g.client = nil
		c.close()
		g.setAwaiting(false)
		text := r.Description
		if text == "" {
			text = r.ErrorType
		}
		g.setStatus(text, true)
		g.signIn.SetSensitive(true) // try again
	}
}

// isPasswordPrompt reports whether a prompt is PAM's plain "Password:",
// which the entry's placeholder already says.
func isPasswordPrompt(s string) bool {
	s = strings.TrimSuffix(strings.TrimSpace(s), ":")
	return strings.EqualFold(s, "password") || strings.EqualFold(s, "password:")
}

// setAwaiting opens the entry when greetd waits for what the user types.
func (g *greeter) setAwaiting(v bool) {
	g.awaiting = v
	g.pass.SetSensitive(v)
	g.signIn.SetSensitive(v)
	if v {
		g.pass.GrabFocus()
	}
}

// setStatus shows what greetd said, or nothing.
func (g *greeter) setStatus(text string, bad bool) {
	g.status.SetText(text)
	g.status.SetVisible(text != "")
	if bad {
		g.status.AddCssClass("error")
	} else {
		g.status.RemoveCssClass("error")
	}
}

// submit sends what was typed. With nothing running it starts over, so a
// failed attempt can be tried again.
func (g *greeter) submit() {
	if g.demo {
		g.setStatus(fmt.Sprintf("demo — %s · %s", g.user.Name, g.session.Name), false)
		return
	}
	if !g.awaiting {
		if g.client == nil {
			g.startAuth()
		}
		return
	}
	secret := g.pass.GetText()
	g.pass.SetText("")
	g.setAwaiting(false)
	g.setStatus(g.i18n.T("Checking…"), false)
	c := g.client
	if c == nil {
		g.startAuth()
		return
	}
	go func() {
		if err := c.post(secret); err != nil {
			post(func() { g.authClosed(c, err) })
		}
	}()
}

// --- power ---

// powerClicked asks logind for the action on the button (greetd has no
// power commands of its own). Without permission the error is shown.
func (g *greeter) powerClicked(b gtk.Button) {
	action := b.GetName()
	switch action {
	case "Suspend":
		g.setStatus(g.i18n.T("Suspending…"), false)
	case "Reboot":
		g.setStatus(g.i18n.T("Restarting…"), false)
	case "PowerOff":
		g.setStatus(g.i18n.T("Shutting down…"), false)
	}
	go func() {
		err := logindAction(action)
		post(func() {
			if err != nil {
				logf("logind %s: %v", action, err)
				g.setStatus(g.i18n.T("No permission."), true)
			}
		})
	}()
}

// logindAction calls org.freedesktop.login1.Manager.<action>.
func logindAction(action string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()
	obj := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
	return obj.Call("org.freedesktop.login1.Manager."+action, 0, false).Err
}
