package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/mediautil"
	"github.com/nbd-wtf/go-nostr"
)

// File contains the filename and a callback to open the file that's called
// asynchronously.
type File struct {
	Name string
	Type string // MIME type
	Size int64
	Open func() (io.ReadCloser, error)
}

type ComposerView struct {
	*gtk.Box
	Input        *Input
	Placeholder  *gtk.Label
	UploadTray   *UploadTray
	EmojiChooser *gtk.EmojiChooser

	ctx   context.Context
	ctrl  *MessagesView
	group *global.Group

	rightBox    *gtk.Box
	emojiButton *gtk.MenuButton
	sendButton  *gtk.Button

	leftBox      *gtk.Box
	uploadButton *gtk.Button

	chooser    *gtk.FileChooserNative
	replyingTo string
}

const (
	sendIcon   = "paper-plane-symbolic"
	emojiIcon  = "sentiment-satisfied-symbolic"
	stopIcon   = "edit-clear-all-symbolic"
	replyIcon  = "mail-reply-sender-symbolic"
	uploadIcon = "list-add-symbolic"
)

func NewComposerView(ctx context.Context, messagesView *MessagesView, group *global.Group) *ComposerView {
	v := &ComposerView{
		ctx:   ctx,
		ctrl:  messagesView,
		group: group,
	}

	v.Input = NewInput(ctx, v, group.Address)

	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroll.SetPropagateNaturalHeight(true)
	scroll.SetMaxContentHeight(1000)
	scroll.SetChild(v.Input)

	v.Placeholder = gtk.NewLabel("")
	v.Placeholder.AddCSSClass("mx-2")
	v.Placeholder.AddCSSClass("px-4")
	v.Placeholder.AddCSSClass("py-2")
	v.Placeholder.AddCSSClass("text-subtle")
	v.Placeholder.SetVAlign(gtk.AlignStart)
	v.Placeholder.SetHAlign(gtk.AlignFill)
	v.Placeholder.SetXAlign(0)
	v.Placeholder.SetEllipsize(pango.EllipsizeEnd)

	revealer := gtk.NewRevealer()
	revealer.SetChild(v.Placeholder)
	revealer.SetCanTarget(false)
	revealer.SetRevealChild(true)
	revealer.SetTransitionType(gtk.RevealerTransitionTypeCrossfade)
	revealer.SetTransitionDuration(75)

	overlay := gtk.NewOverlay()
	overlay.SetChild(scroll)
	overlay.AddOverlay(revealer)
	overlay.SetClipOverlay(revealer, true)

	// Show or hide the placeholder when the buffer is empty or not.
	updatePlaceholderVisibility := func() {
		start, end := v.Input.Buffer.Bounds()
		// Reveal if the buffer has 0 length.
		revealer.SetRevealChild(start.Offset() == end.Offset())
	}
	v.Input.Buffer.ConnectChanged(updatePlaceholderVisibility)
	updatePlaceholderVisibility()

	v.UploadTray = NewUploadTray()

	middle := gtk.NewBox(gtk.OrientationVertical, 0)
	middle.Append(overlay)
	middle.Append(v.UploadTray)

	v.uploadButton = newActionButton(actionButtonData{
		Name: "Upload File",
		Icon: uploadIcon,
		Func: v.upload,
	})

	v.leftBox = gtk.NewBox(gtk.OrientationHorizontal, 0)

	v.EmojiChooser = gtk.NewEmojiChooser()
	v.EmojiChooser.ConnectEmojiPicked(func(emoji string) { v.insertEmoji(emoji) })

	v.emojiButton = gtk.NewMenuButton()
	v.emojiButton.SetIconName(emojiIcon)
	v.emojiButton.SetVAlign(gtk.AlignCenter)
	v.emojiButton.SetTooltipText("Choose Emoji")
	v.emojiButton.SetPopover(v.EmojiChooser)

	v.sendButton = gtk.NewButtonFromIconName(sendIcon)
	v.sendButton.SetVAlign(gtk.AlignCenter)
	v.sendButton.SetTooltipText("Send Message")
	v.sendButton.SetHasFrame(false)
	v.sendButton.ConnectClicked(v.publish)

	v.rightBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	v.rightBox.SetHAlign(gtk.AlignEnd)

	v.resetAction()

	v.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	v.Box.SetVAlign(gtk.AlignEnd)
	v.Box.Append(v.leftBox)
	v.Box.Append(middle)
	v.Box.Append(v.rightBox)

	v.SetPlaceholderMarkup("")

	return v
}

// SetPlaceholder sets the composer's placeholder. The default is used if an
// empty string is given.
func (v *ComposerView) SetPlaceholderMarkup(markup string) {
	if markup == "" {
		v.ResetPlaceholder()
		return
	}

	v.Placeholder.SetMarkup(markup)
}

func (v *ComposerView) ResetPlaceholder() {
	v.Placeholder.SetText("Message " + v.group.Address.String()) // + gtkcord.ChannelNameFromID(v.ctx, v.chID))
}

// actionButton is a button that is used in the composer bar.
type actionButton interface {
	newButton() gtk.Widgetter
}

// existingActionButton is a button that already exists in the composer bar.
type existingActionButton struct{ gtk.Widgetter }

func (a existingActionButton) newButton() gtk.Widgetter { return a }

// actionButtonData is the data that the action button in the composer bar is
// currently doing.
type actionButtonData struct {
	Name locale.Localized
	Icon string
	Func func()
}

func newActionButton(a actionButtonData) *gtk.Button {
	button := gtk.NewButton()
	button.SetHasFrame(false)
	button.SetHAlign(gtk.AlignCenter)
	button.SetVAlign(gtk.AlignCenter)
	button.SetSensitive(a.Func != nil)
	button.SetIconName(a.Icon)
	button.SetTooltipText(a.Name.String())
	button.ConnectClicked(func() { a.Func() })

	return button
}

func (a actionButtonData) newButton() gtk.Widgetter {
	return newActionButton(a)
}

type actions struct {
	left  []actionButton
	right []actionButton
}

// setAction sets the action of the button in the composer.
func (v *ComposerView) setActions(actions actions) {
	gtkutil.RemoveChildren(v.leftBox)
	gtkutil.RemoveChildren(v.rightBox)

	for _, a := range actions.left {
		v.leftBox.Append(a.newButton())
	}
	for _, a := range actions.right {
		v.rightBox.Append(a.newButton())
	}
}

func (v *ComposerView) resetAction() {
	v.setActions(actions{
		left:  []actionButton{existingActionButton{v.uploadButton}},
		right: []actionButton{existingActionButton{v.emojiButton}, existingActionButton{v.sendButton}},
	})
}

func (v *ComposerView) upload() {
	// From GTK's documentation:
	//   Note that unlike GtkDialog, GtkNativeDialog objects are not toplevel
	//   widgets, and GTK does not keep them alive. It is your responsibility to
	//   keep a reference until you are done with the object.
	v.chooser = gtk.NewFileChooserNative(
		"Upload Files",
		app.GTKWindowFromContext(v.ctx),
		gtk.FileChooserActionOpen,
		"Upload", "Cancel",
	)
	v.chooser.SetSelectMultiple(true)
	v.chooser.SetModal(true)
	v.chooser.ConnectResponse(func(resp int) {
		if resp == int(gtk.ResponseAccept) {
			v.addFiles(v.chooser.Files())
		}
		v.chooser.Destroy()
		v.chooser = nil
	})
	v.chooser.Show()
}

func (v *ComposerView) addFiles(list gio.ListModeller) {
	go func() {
		var i uint
		for v.ctx.Err() == nil {
			obj := list.Item(i)
			if obj == nil {
				break
			}

			file := obj.Cast().(gio.Filer)
			path := file.Path()

			f := File{
				Name: file.Basename(),
				Type: mediautil.FileMIME(v.ctx, file),
				Size: mediautil.FileSize(v.ctx, file),
			}

			if path != "" {
				f.Open = func() (io.ReadCloser, error) {
					return os.Open(path)
				}
			} else {
				f.Open = func() (io.ReadCloser, error) {
					r, err := file.Read(v.ctx)
					if err != nil {
						return nil, err
					}
					return gioutil.Reader(v.ctx, r), nil
				}
			}

			glib.IdleAdd(func() { v.UploadTray.AddFile(f) })
			i++
		}
	}()
}

func (v *ComposerView) peekContent() (string, []File) {
	start, end := v.Input.Buffer.Bounds()
	text := v.Input.Buffer.Text(start, end, false)
	files := v.UploadTray.Files()
	return text, files
}

func (v *ComposerView) commitContent() (string, []File) {
	start, end := v.Input.Buffer.Bounds()
	text := v.Input.Buffer.Text(start, end, false)
	v.Input.Buffer.Delete(start, end)
	files := v.UploadTray.Clear()
	return text, files
}

func (v *ComposerView) insertEmoji(emoji string) {
	endIter := v.Input.Buffer.EndIter()
	v.Input.Buffer.Insert(endIter, emoji)
}

func (v *ComposerView) publish() {
	text, files := v.commitContent()
	if text == "" && len(files) == 0 {
		return
	}

	err := v.ctrl.currentGroup.SendChatMessage(v.ctx, text, v.replyingTo)
	if err != nil {
		slog.Warn(err.Error())
		win.ErrorToast(strings.Replace(err.Error(), " msg: ", " ", 1))
		return
	}

	v.ctrl.stopEditingOrReplying()
}

// textBufferIsReaction returns whether the text buffer is for adding a reaction.
// It is true if the input matches something like "+<emoji>".
func textBufferIsReaction(buffer string) bool {
	buffer = strings.TrimRightFunc(buffer, unicode.IsSpace)
	return strings.HasPrefix(buffer, "+") && !strings.ContainsFunc(buffer, unicode.IsSpace)
}

// StartReplyingTo starts replying to the given message. Visually, there is no
// difference except for the send button being different.
func (v *ComposerView) StartReplyingTo(msg *nostr.Event) {
	v.ctrl.stopEditingOrReplying()
	v.replyingTo = msg.ID

	v.SetPlaceholderMarkup(fmt.Sprintf(
		"Replying to %s",
		msg.ID,
	))

	// mentionToggle := gtk.NewToggleButton()
	// mentionToggle.SetIconName("online-symbolic")
	// mentionToggle.SetHasFrame(false)
	// mentionToggle.SetActive(true)
	// mentionToggle.SetHAlign(gtk.AlignCenter)
	// mentionToggle.SetVAlign(gtk.AlignCenter)
	// mentionToggle.ConnectToggled(func() {
	// 	if mentionToggle.Active() {
	// 		v.state.replying = replyingMention
	// 	} else {
	// 		v.state.replying = replyingNoMention
	// 	}
	// })

	// v.setActions(actions{
	// 	left: []actionButton{
	// 		existingActionButton{v.uploadButton},
	// 	},
	// 	right: []actionButton{
	// 		existingActionButton{v.emojiButton},
	// 		existingActionButton{mentionToggle},
	// 		actionButtonData{
	// 			Name: "Reply",
	// 			Icon: replyIcon,
	// 			Func: v.publish,
	// 		},
	// 	},
	// })
}

func (v *ComposerView) StopReplying() {
	v.ctrl.stopEditingOrReplying()
	v.replyingTo = ""
	v.SetPlaceholderMarkup("")
	v.RemoveCSSClass("composer-replying")
	v.resetAction()
}
