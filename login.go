package main

import (
	"context"
	"log"
	"log/slog"
	"strings"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/utils/secret"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type LoginPage struct {
	*gtk.Box
	ctx context.Context

	driver     secret.Driver
	errorLabel *gtk.Label
}

func NewLoginPage(ctx context.Context, w *Window) *LoginPage {
	p := LoginPage{
		ctx: ctx,
	}

	header := adw.NewHeaderBar()
	header.SetShowBackButton(false)
	header.SetShowTitle(true)

	input := adw.NewEntryRow()
	password := adw.NewPasswordEntryRow()

	login := func() {
		err := p.login(input.Text(), password.Text())
		if err != nil {
			return
		}

		// here we have a signer, so we can store our input value
		p.driver.Set("key-or-bunker", []byte(input.Text()))
	}

	input.SetTitle("nsec, ncryptsec or bunker")
	input.ConnectChanged(func() {
		if strings.HasPrefix(input.Text(), "ncryptsec1") {
			password.Show()
		}
	})
	input.AddCSSClass("mb-4")
	input.AddCSSClass("rounded")
	input.ConnectActivate(login)
	input.Show()

	password.SetTitle("password")
	password.ConnectActivate(login)
	password.AddCSSClass("mb-4")
	password.AddCSSClass("rounded")
	password.Hide()

	submit := gtk.NewButtonWithLabel("Log In")
	submit.AddCSSClass("suggested-action")
	submit.SetHExpand(true)
	submit.ConnectClicked(login)

	body := gtk.NewListBox()
	body.SetHAlign(gtk.AlignCenter)
	body.SetVAlign(gtk.AlignCenter)
	body.SetVExpand(true)
	body.SetHExpand(true)
	body.Show()

	p.errorLabel = gtk.NewLabel("")
	p.errorLabel.SetHAlign(gtk.AlignStart)
	p.errorLabel.SetMarginTop(10)
	p.errorLabel.SetMarginBottom(10)
	p.errorLabel.SetMarginStart(10)
	p.errorLabel.SetMarginEnd(10)
	p.errorLabel.Hide()

	body.Append(p.errorLabel)
	body.Append(input)
	body.Append(password)
	body.Append(submit)

	p.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	p.Box.Append(header)
	p.Box.Append(body)

	return &p
}

func (p *LoginPage) login(input, password string) error {
	err := global.Init(p.ctx, input, password)
	if err != nil {
		slog.Error("error initializing signer", err)
		p.errorLabel.SetText(err.Error())
		p.errorLabel.Show()
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
			}
		}

		return func() {
			value := string(b)
			if !strings.HasPrefix(value, "ncryptsec1") {
				p.login(value, "")
			}
		}
	})
}
