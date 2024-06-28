package main

import (
	"context"
	"log"
	"log/slog"
	"strings"

	"fiatjaf.com/nostr-gtk/secret"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type LoginPage struct {
	*gtk.Box
	ctx        context.Context
	driver     secret.Driver
	input      *adw.EntryRow
	password   *adw.PasswordEntryRow
	errorLabel *gtk.Label
}

func NewLoginPage(ctx context.Context, w *Window) *LoginPage {
	p := LoginPage{
		ctx: ctx,
	}

	header := adw.NewHeaderBar()
	header.SetShowBackButton(false)
	header.SetShowTitle(true)

	p.input = adw.NewEntryRow()
	p.password = adw.NewPasswordEntryRow()

	login := func() {
		err := p.login(p.input.Text(), p.password.Text())
		if err != nil {
			return
		}

		// here we have a signer, so we can store our input value
		p.driver.Set("key-or-bunker", []byte(p.input.Text()))
	}

	p.input.SetTitle("nsec, ncryptsec or bunker")
	p.input.ConnectChanged(func() {
		if strings.HasPrefix(p.input.Text(), "ncryptsec1") {
			p.password.Show()
		}
	})
	p.input.AddCSSClass("mb-4")
	p.input.AddCSSClass("rounded")
	p.input.SetSizeRequest(400, -1)
	p.input.SetHExpand(false)
	p.input.ConnectEntryActivated(login)
	p.input.Show()

	p.password.SetTitle("password")
	p.password.AddCSSClass("mb-4")
	p.password.AddCSSClass("rounded")
	p.password.ConnectEntryActivated(login)
	p.password.Hide()

	submit := gtk.NewButtonWithLabel("Log In")
	submit.AddCSSClass("suggested-action")
	submit.SetHExpand(false)
	submit.ConnectClicked(login)

	p.errorLabel = gtk.NewLabel("")
	p.errorLabel.SetHAlign(gtk.AlignStart)
	p.errorLabel.AddCSSClass("p-4")
	p.errorLabel.AddCSSClass("rounded")
	p.errorLabel.AddCSSClass("alert")
	p.errorLabel.SetWrap(true)
	p.errorLabel.Hide()

	wrapper := gtk.NewBox(gtk.OrientationVertical, 10)
	wrapper.SetSizeRequest(400, -1)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetVAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(true)

	wrapper.Append(p.errorLabel)
	wrapper.Append(p.input)
	wrapper.Append(p.password)
	wrapper.Append(submit)
	wrapper.Show()

	p.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	p.Box.Append(header)
	p.Box.Append(wrapper)

	return &p
}

func (p *LoginPage) login(input, password string) error {
	err := global.Init(p.ctx, input, password)
	if err != nil {
		slog.Error("error initializing signer", err)
		p.errorLabel.SetText(err.Error())
		p.errorLabel.Show()
		if !strings.HasPrefix(input, "ncryptsec1") {
			p.input.GrabFocus()
		} else {
			p.password.GrabFocus()
		}
		return err
	}

	// switch to chat page
	win.Stack.SetVisibleChild(win.main)
	win.main.Groups.switchTo(nip29.GroupAddress{})
	win.SetTitle("Chat")
	return nil
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
				p.input.GrabFocus()
			}
		}

		return func() {
			value := string(b)
			p.input.SetText(value)
			if !strings.HasPrefix(value, "ncryptsec1") {
				p.login(value, "")
			} else {
				p.password.GrabFocus()
			}
		}
	})
}
