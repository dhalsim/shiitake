package window

import (
	"context"
	"log"
	"strings"

	"fiatjaf.com/shiitake/components/form_entry"
	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/window/loading"
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// LoginLoginComponent is the main component in the login page.
type LoginComponent struct {
	*gtk.Box
	Inner *gtk.Box

	Loading     *loading.PulsatingBar
	KeyOrBunker *form_entry.FormEntry
	Bottom      *gtk.Box
	ErrorRev    *gtk.Revealer
	Submit      *gtk.Button

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

	c.KeyOrBunker = form_entry.New("nsec, ncryptsec or bunker")
	c.KeyOrBunker.FocusNextOnActivate()
	c.KeyOrBunker.Entry.SetInputPurpose(gtk.InputPurposeEmail)
	c.KeyOrBunker.ConnectActivate(c.ForceSubmit)

	c.ErrorRev = gtk.NewRevealer()
	c.ErrorRev.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	c.ErrorRev.SetRevealChild(false)

	c.Submit = gtk.NewButtonWithLabel("Log In")
	c.Submit.AddCSSClass("suggested-action")
	c.Submit.AddCSSClass("login-button")
	c.Submit.SetHExpand(true)
	c.Submit.ConnectClicked(c.handleSubmit)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	buttonBox.Append(c.Submit)

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

func (c *LoginComponent) ForceSubmit() {
	c.Submit.Activate()
}

// TryLoginFromDriver loads a secret from the keyring or filesystem and tries to login with it
func (c *LoginComponent) TryLoginFromDriver() {
	c.Loading.Show()
	c.SetSensitive(false)

	done := func() {
		c.Loading.Hide()
		c.SetSensitive(true)
	}

	gtkutil.Async(c.ctx, func() func() {
		b, err := c.page.driver.Get("key-or-bunker")
		if err != nil {
			log.Println("key-or-bunker not found from driver:", err)
			return done
		}

		value := string(b)
		c.loginWithInput(value)

		return func() {
			done()
		}
	})
}

func (c *LoginComponent) handleSubmit() {
	c.loginWithInput(c.KeyOrBunker.Entry.Text())
}

func (c *LoginComponent) loginWithInput(input string) {
	log.Printf("using '%s'\n", input)
	if strings.HasPrefix(input, "ncryptsec1") {
		promptPassword(c.ctx, func(ok bool, password string) {
			c.loginWithPassword(input, password)
		})
	} else {
		c.loginWithPassword(input, "")
	}
}

func (c *LoginComponent) loginWithPassword(input string, password string) {
	// set busy
	c.SetSensitive(false)
	c.Loading.Show()

	err := global.Init(c.ctx, input, password)
	if err != nil {
		c.SetSensitive(true)
		c.Loading.Hide()
		log.Println("error initializing signer", err)
		return
	}

	// here we have a signer, so we can store our input value
	c.page.driver.Set("key-or-bunker", []byte(input))

	// set done
	c.SetSensitive(true)
	c.Loading.Hide()

	// trigger app start
	c.page.w.OnLogin()
}
