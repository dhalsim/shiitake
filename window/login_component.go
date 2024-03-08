package window

import (
	"context"
	"log"
	"strings"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/window/loading"
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	sdk "github.com/nbd-wtf/nostr-sdk"
)

// LoginLoginComponent is the main component in the login page.
type LoginComponent struct {
	*gtk.Box
	Inner *gtk.Box

	Loading     *loading.PulsatingBar
	KeyOrBunker *FormEntry
	Bottom      *gtk.Box
	ErrorRev    *gtk.Revealer
	LogIn       *gtk.Button

	ctx  context.Context
	page *LoginPage
}

var componentCSS = cssutil.Applier("login-component", `
	.login-component {
		background: mix(@theme_bg_color, @theme_fg_color, 0.05);
		border-radius: 12px;
		min-width: 250px;
		margin:  12px;
		padding: 0;
	}
	.login-component > *:not(.osd) {
		margin: 0 8px;
	}
	.login-component > *:nth-child(2) {
		margin-top: 6px;
	}
	.login-component > *:first-child {
		margin-top: 8px;
	}
	.login-component > *:not(:first-child) {
		margin-bottom: 4px;
	}
	.login-component > *:last-child {
		margin-bottom: 8px;
	}
	.login-component > notebook {
		background: none;
	}
	.login-component .adaptive-errorlabel {
		margin-bottom: 8px;
	}
	.login-button {
		background-color: #7289DA;
		color: #FFFFFF;
	}
	.login-with {
		font-weight: bold;
		margin-bottom: 2px;
	}
	.login-decrypt-button {
		margin-left: 4px;
	}
`)

// NewLoginComponent creates a new login LoginComponent.
func NewLoginComponent(ctx context.Context, p *LoginPage) *LoginComponent {
	c := LoginComponent{
		ctx:  ctx,
		page: p,
	}

	c.Loading = loading.NewPulsatingBar(loading.PulseFast | loading.PulseBarOSD)

	loginWith := gtk.NewLabel("Login with nsec or ncryptsec:")
	loginWith.AddCSSClass("login-with")
	loginWith.SetXAlign(0)

	c.KeyOrBunker = NewFormEntry("nsec, ncryptsec or bunker")
	c.KeyOrBunker.FocusNextOnActivate()
	c.KeyOrBunker.Entry.SetInputPurpose(gtk.InputPurposeEmail)
	c.KeyOrBunker.ConnectActivate(c.Login)

	c.ErrorRev = gtk.NewRevealer()
	c.ErrorRev.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	c.ErrorRev.SetRevealChild(false)

	c.LogIn = gtk.NewButtonWithLabel("Log In")
	c.LogIn.AddCSSClass("suggested-action")
	c.LogIn.AddCSSClass("login-button")
	c.LogIn.SetHExpand(true)
	c.LogIn.ConnectClicked(c.login)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	buttonBox.Append(c.LogIn)

	c.Inner = gtk.NewBox(gtk.OrientationVertical, 0)
	c.Inner.Append(loginWith)
	c.Inner.Append(c.KeyOrBunker)
	c.Inner.Append(c.ErrorRev)
	c.Inner.Append(buttonBox)
	componentCSS(c.Inner)

	c.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	c.Box.AddCSSClass("login-component-outer")
	c.Box.SetHAlign(gtk.AlignCenter)
	c.Box.SetVAlign(gtk.AlignCenter)
	c.Box.Append(c.Loading)
	c.Box.Append(c.Inner)

	return &c
}

// ShowError reveals the error label and shows it to the user.
func (c *LoginComponent) ShowError(err error) {
	errLabel := adaptive.NewErrorLabel(err)
	c.ErrorRev.SetChild(errLabel)
	c.ErrorRev.SetRevealChild(true)
}

// HideError hides the error label.
func (c *LoginComponent) HideError() {
	c.ErrorRev.SetRevealChild(false)
}

// Login presses the Login button.
func (c *LoginComponent) Login() {
	c.LogIn.Activate()
}

func (c *LoginComponent) login() {
	value := c.KeyOrBunker.Entry.Text()

	if strings.HasPrefix(value, "ncryptsec1") {
		promptPassword(c.ctx, func(ok bool, password string) {
			c.loginWithPassword(password)
		})
	} else {
		c.loginWithPassword("")
	}
}

func (c *LoginComponent) loginWithPassword(password string) {
	// set busy
	c.SetSensitive(false)
	c.Loading.Show()

	value := c.KeyOrBunker.Entry.Text()
	opts := &sdk.SignerOptions{Password: password}

	if err := global.Sys.InitSigner(c.ctx, value, opts); err != nil {
		log.Println(err)
		// TODO: display error
		return
	}

	// here we have a signer, so we can store our input value
	c.page.driver.Set("key-or-bunker", []byte(value))

	// set done
	c.SetSensitive(true)
	c.Loading.Hide()

	// trigger app start
	c.page.w.OnLogin()
}

// FormEntry is a widget containing a label and an entry.
type FormEntry struct {
	*gtk.Box
	Label *gtk.Label
	Entry *gtk.Entry
}

var formEntryCSS = cssutil.Applier("login-formentry", ``)

// NewFormEntry creates a new FormEntry.
func NewFormEntry(label string) *FormEntry {
	e := FormEntry{}
	e.Label = gtk.NewLabel(label)
	e.Label.SetXAlign(0)

	e.Entry = gtk.NewEntry()
	e.Entry.SetVExpand(true)
	e.Entry.SetHasFrame(true)

	e.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	e.Box.Append(e.Label)
	e.Box.Append(e.Entry)
	formEntryCSS(e)

	return &e
}

// Text gets the value entry.
func (e *FormEntry) Text() string { return e.Entry.Text() }

// FocusNext navigates to the next widget.
func (e *FormEntry) FocusNext() {
	e.Entry.Emit("move-focus", gtk.DirTabForward)
}

// FocusNextOnActivate binds Enter to navigate to the next widget when it's
// pressed.
func (e *FormEntry) FocusNextOnActivate() {
	e.Entry.ConnectActivate(e.FocusNext)
}

// ConnectActivate connects the activate signal hanlder to the Entry.
func (e *FormEntry) ConnectActivate(f func()) {
	e.Entry.ConnectActivate(f)
}
