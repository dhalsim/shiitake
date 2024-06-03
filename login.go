package main

import (
	"context"
	"log"
	"log/slog"
	"strings"

	"fiatjaf.com/shiitake/components/form_entry"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/chatkit/kits/secret"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type LoginPage struct {
	*gtk.Box
	Body *gtk.Box

	ctx context.Context

	driver secret.Driver

	KeyOrBunker *form_entry.FormEntry
	ErrorRev    *gtk.Revealer
}

func NewLoginPage(ctx context.Context, w *Window) *LoginPage {
	p := LoginPage{
		ctx: ctx,
	}

	header := gtk.NewHeaderBar()
	header.SetShowTitleButtons(true)

	loginWith := gtk.NewLabel("Login with nsec or ncryptsec:")
	loginWith.SetXAlign(0)

	submit := gtk.NewButtonWithLabel("Log In")
	submit.SetHExpand(true)
	submit.ConnectClicked(func() {
		p.loginWithInput(p.KeyOrBunker.Entry.Text())
	})

	p.KeyOrBunker = form_entry.New("nsec, ncryptsec or bunker")
	p.KeyOrBunker.FocusNextOnActivate()
	p.KeyOrBunker.Entry.SetInputPurpose(gtk.InputPurposeEmail)
	p.KeyOrBunker.ConnectActivate(func() {
		submit.Activate()
	})

	p.ErrorRev = gtk.NewRevealer()
	p.ErrorRev.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	p.ErrorRev.SetRevealChild(false)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	buttonBox.Append(submit)

	form := gtk.NewBox(gtk.OrientationVertical, 0)
	form.Append(loginWith)
	form.Append(p.KeyOrBunker)
	form.Append(p.ErrorRev)
	form.Append(buttonBox)

	p.Body = gtk.NewBox(gtk.OrientationVertical, 0)
	p.Body.SetHAlign(gtk.AlignCenter)
	p.Body.SetVAlign(gtk.AlignCenter)
	p.Body.SetVExpand(true)
	p.Body.SetHExpand(true)
	p.Body.SetSensitive(true)
	p.Body.Show()
	p.Body.Append(form)

	p.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	p.Box.Append(header)
	p.Box.Append(p.Body)

	return &p
}

func (p *LoginPage) TryLoginFromDriver(ctx context.Context) {
	gtkutil.Async(p.ctx, func() func() {
		if keyring := secret.KeyringDriver(ctx); keyring != nil && keyring.IsAvailable() {
			p.driver = keyring
		} else {
			p.driver = secret.PlainFileDriver()
		}
		b, err := p.driver.Get("key-or-bunker")
		if err != nil {
			return func() {
				log.Println("key-or-bunker not found from driver:", err)
				// display login form
				win.Stack.SetVisibleChild(p)
				win.SetTitle("Login")
			}
		}

		return func() {
			value := string(b)
			p.loginWithInput(value)
		}
	})
}

func (p *LoginPage) ShowError(err error) {
	errLabel := adaptive.NewErrorLabel(err)
	p.ErrorRev.SetChild(errLabel)
	p.ErrorRev.SetRevealChild(true)
}

func (p *LoginPage) hideError() {
	p.ErrorRev.SetRevealChild(false)
}

func (p *LoginPage) loginWithInput(input string) {
	log.Printf("using '%s'\n", input)
	if strings.HasPrefix(input, "ncryptsec1") {
		promptPassword(p.ctx, func(ok bool, password string) {
			p.loginWithPassword(input, password)
		})
	} else {
		p.loginWithPassword(input, "")
	}
}

func (p *LoginPage) loginWithPassword(input string, password string) {
	// set busy
	p.Body.SetSensitive(false)

	err := global.Init(p.ctx, input, password)
	if err != nil {
		p.Body.SetSensitive(true)
		slog.Error("error initializing signer", err)
		return
	}

	// here we have a signer, so we can store our input value
	p.driver.Set("key-or-bunker", []byte(input))

	// set done
	p.SetSensitive(true)

	// switch to chat page
	win.Stack.SetVisibleChild(win.main)
	win.main.messagesView.switchTo(nip29.GroupAddress{})
	win.SetTitle("Chat")
}
