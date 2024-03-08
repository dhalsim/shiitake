package utils

import (
	"fmt"

	"fiatjaf.com/shiitake/window"
	"github.com/diamondburned/gotkit/app/notify"
)

func sendNotification(w window.Window) {
	notify.Send(w.ctx, notify.Notification{
		ID: notify.HashID(ev.ChannelID),
		Title: fmt.Sprintf(
			"%s (%s)",
			state.AuthorDisplayName(ev),
			gtkcord.ChannelNameFromID(w.ctx, ev.ChannelID),
		),
		Body:  ev.Message.ChannelID.String(),
		Icon:  notify.IconURL(w.ctx, global.InjectAvatarSize(ev.Author.AvatarURL()), notify.IconName("avatar-default-symbolic")),
		Sound: notify.MessageSound,
		Action: notify.Action{
			ActionID: "app.open-channel",
			Argument: gtkcord.NewChannelIDVariant(ev.ChannelID),
		},
	})
}
