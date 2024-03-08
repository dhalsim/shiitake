package window

import (
	"context"
	"log"

	"github.com/diamondburned/chatkit/kits/secret"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// LoginPage is the page containing the login forms.
type LoginPage struct {
	*gtk.Box
	Header *gtk.HeaderBar
	Login  *LoginComponent

	driver secret.Driver

	ctx context.Context
	w   *Window
}

var pageCSS = cssutil.Applier("login-page", ``)

// NewLoginPage creates a new LoginPage.
func NewLoginPage(ctx context.Context, w *Window) *LoginPage {
	p := LoginPage{
		ctx: ctx,
		w:   w,
	}

	if keyring := secret.KeyringDriver(ctx); keyring.IsAvailable() {
		p.driver = keyring
	} else {
		p.driver = secret.PlainFileDriver()
	}

	p.Header = gtk.NewHeaderBar()
	p.Header.AddCSSClass("login-page-header")
	p.Header.SetShowTitleButtons(true)

	p.Login = NewLoginComponent(ctx, &p)
	p.Login.SetVExpand(true)
	p.Login.SetHExpand(true)

	p.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	p.Box.Append(p.Header)
	p.Box.Append(p.Login)
	pageCSS(p)

	return &p
}

func (p *LoginPage) LoadSecret() {
	p.Login.Loading.Show()
	p.Login.SetSensitive(false)

	done := func() {
		p.Login.Loading.Hide()
		p.Login.SetSensitive(true)
	}

	gtkutil.Async(p.ctx, func() func() {
		b, err := p.driver.Get("key-or-bunker")
		if err != nil {
			log.Println("key-or-bunker not found from driver:", err)
			return done
		}
		log.Println("loaded", string(b))

		return func() {
			done()
			p.w.OnLogin()
		}
	})
}
