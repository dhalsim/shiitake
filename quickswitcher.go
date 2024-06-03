package main

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/textutil"
	"github.com/nbd-wtf/go-nostr/nip29"
	"github.com/sahilm/fuzzy"
)

// QuickSwitcher is a search box capable of looking up guilds and channels for
// quickly jumping to them. It replicates the Ctrl+K dialog of the desktop
// client.
type QuickSwitcher struct {
	*gtk.Box
	ctx   gtkutil.Cancellable
	text  string
	index qwIndex

	search     *gtk.SearchEntry
	chosenFunc func()

	entryScroll *gtk.ScrolledWindow
	entryList   *gtk.ListBox
	entries     []qwEntry
}

type qwEntry struct {
	*gtk.ListBoxRow
	indexItem qwIndexItem
}

func NewQuickSwitcher(ctx context.Context) *QuickSwitcher {
	var qs QuickSwitcher
	qs.index.update(ctx)

	qs.search = gtk.NewSearchEntry()
	qs.search.SetHExpand(true)
	qs.search.SetObjectProperty("placeholder-text", "Search")
	qs.search.ConnectActivate(func() { qs.selectEntry() })
	qs.search.ConnectNextMatch(func() { qs.moveDown() })
	qs.search.ConnectPreviousMatch(func() { qs.moveUp() })
	qs.search.ConnectSearchChanged(func() {
		qs.text = qs.search.Text()
		qs.do()
	})

	if qs.search.ObjectProperty("search-delay") != nil {
		// Only GTK v4.8 and onwards.
		qs.search.SetObjectProperty("search-delay", 100)
	}

	keyCtrl := gtk.NewEventControllerKey()
	keyCtrl.ConnectKeyPressed(func(val, _ uint, state gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Up:
			return qs.moveUp()
		case gdk.KEY_Down, gdk.KEY_Tab:
			return qs.moveDown()
		default:
			return false
		}
	})
	qs.search.AddController(keyCtrl)

	qs.entryList = gtk.NewListBox()
	qs.entryList.SetVExpand(true)
	qs.entryList.SetSelectionMode(gtk.SelectionSingle)
	qs.entryList.SetActivateOnSingleClick(true)
	qs.entryList.SetPlaceholder(qsListPlaceholder())
	qs.entryList.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		qs.choose(row.Index())
	})

	entryViewport := gtk.NewViewport(nil, nil)
	entryViewport.SetScrollToFocus(true)
	entryViewport.SetChild(qs.entryList)

	qs.entryScroll = gtk.NewScrolledWindow()
	qs.entryScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	qs.entryScroll.SetChild(entryViewport)
	qs.entryScroll.SetVExpand(true)

	qs.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	qs.Box.SetVExpand(true)
	qs.Box.Append(qs.search)
	qs.Box.Append(qs.entryScroll)

	qs.ctx = gtkutil.WithVisibility(ctx, qs.search)
	qs.search.SetKeyCaptureWidget(qs)

	return &qs
}

func qsListLoading() gtk.Widgetter {
	loading := gtk.NewSpinner()
	loading.SetSizeRequest(24, 24)
	loading.SetVAlign(gtk.AlignCenter)
	loading.SetHAlign(gtk.AlignCenter)
	loading.Start()
	return loading
}

func qsListPlaceholder() gtk.Widgetter {
	l := gtk.NewLabel("Where would you like to go?")
	l.SetAttributes(textutil.Attrs(
		pango.NewAttrScale(1.15),
	))
	l.SetVAlign(gtk.AlignCenter)
	l.SetHAlign(gtk.AlignCenter)
	return l
}

func (qs *QuickSwitcher) do() {
	if qs.text == "" {
		return
	}

	for i, e := range qs.entries {
		qs.entryList.Remove(e)
		qs.entries[i] = qwEntry{}
	}
	qs.entries = qs.entries[:0]

	for _, match := range qs.index.search(qs.text) {
		e := qwEntry{
			ListBoxRow: match.Row(qs.ctx.Take()),
			indexItem:  match,
		}

		qs.entries = append(qs.entries, e)
		qs.entryList.Append(e)
	}

	if len(qs.entries) > 0 {
		qs.entryList.SelectRow(qs.entries[0].ListBoxRow)
	}
}

func (qs *QuickSwitcher) choose(n int) {
	entry := qs.entries[n]

	var ok bool
	switch item := entry.indexItem.(type) {
	case qwRelayItem:
		win.OpenRelay(item.url)
	case qwGroupItem:
		win.OpenGroup(item.group.Address)
	}
	if !ok {
		log.Println("quickswitcher: failed to activate action")
	}

	if qs.chosenFunc != nil {
		qs.chosenFunc()
	}
}

// ConnectChosen connects a function to be called when an entry is chosen.
func (qs *QuickSwitcher) ConnectChosen(f func()) {
	if qs.chosenFunc != nil {
		add := f
		old := qs.chosenFunc
		f = func() {
			old()
			add()
		}
	}
	qs.chosenFunc = f
}

func (qs *QuickSwitcher) selectEntry() bool {
	if len(qs.entries) == 0 {
		return false
	}

	row := qs.entryList.SelectedRow()
	if row == nil {
		return false
	}

	qs.choose(row.Index())
	return true
}

func (qs *QuickSwitcher) moveUp() bool   { return qs.move(false) }
func (qs *QuickSwitcher) moveDown() bool { return qs.move(true) }

func (qs *QuickSwitcher) move(down bool) bool {
	if len(qs.entries) == 0 {
		return false
	}

	row := qs.entryList.SelectedRow()
	if row == nil {
		qs.entryList.SelectRow(qs.entries[0].ListBoxRow)
		return true
	}

	ix := row.Index()
	if down {
		ix++
		if ix == len(qs.entries) {
			ix = 0
		}
	} else {
		ix--
		if ix == -1 {
			ix = len(qs.entries) - 1
		}
	}

	qs.entryList.SelectRow(qs.entries[ix].ListBoxRow)

	// Steal focus. This is a hack to scroll to the selected item without having
	// to manually calculate the coordinates.
	var target gtk.Widgetter = qs.search
	if focused := app.WindowFromContext(qs.ctx.Take()).Focus(); focused != nil {
		target = focused
	}
	targetBase := gtk.BaseWidget(target)
	qs.entries[ix].ListBoxRow.GrabFocus()
	targetBase.GrabFocus()

	return true
}

type QuickSwitcherDialog struct {
	*adw.ApplicationWindow
	QuickSwitcher *QuickSwitcher
}

func ShowQuickSwitcherDialog(ctx context.Context) {
	d := NewQuickSwitcherDialog(ctx)
	d.Show()
}

func NewQuickSwitcherDialog(ctx context.Context) *QuickSwitcherDialog {
	qs := NewQuickSwitcher(ctx)
	qs.Box.Remove(qs.search) // jank
	qs.search.SetHExpand(true)

	gwin := app.GTKWindowFromContext(ctx)
	application := app.FromContext(ctx)

	header := adw.NewHeaderBar()
	header.SetTitleWidget(qs.search)

	toolbarView := adw.NewToolbarView()
	toolbarView.SetTopBarStyle(adw.ToolbarFlat)
	toolbarView.AddTopBar(header)
	toolbarView.SetContent(qs)

	d := QuickSwitcherDialog{QuickSwitcher: qs}
	d.ApplicationWindow = adw.NewApplicationWindow(application.Application)
	d.SetTransientFor(gwin)
	d.SetDefaultSize(400, 275)
	d.SetModal(true)
	d.SetDestroyWithParent(true)
	d.SetTitle(application.SuffixedTitle("Quick Switcher"))
	d.SetContent(toolbarView)
	d.ConnectShow(func() {
		qs.search.GrabFocus()
	})

	// SetDestroyWithParent doesn't work for some reason, so we have to manually
	// destroy the QuickSwitcher on transient window destroy.
	gwin.ConnectCloseRequest(func() bool {
		d.Destroy()
		return false
	})

	qs.ConnectChosen(func() {
		d.Close()
	})

	esc := gtk.NewEventControllerKey()
	esc.SetName("dialog-escape")
	esc.ConnectKeyPressed(func(val, _ uint, state gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Escape:
			d.Close()
			return true
		}
		return false
	})

	qs.search.SetKeyCaptureWidget(d)
	qs.search.AddController(esc)

	return &d
}

type qwIndexItem interface {
	Row(context.Context) *gtk.ListBoxRow
	String() string
}

type qwIndexItems []qwIndexItem

func (its qwIndexItems) String(i int) string { return its[i].String() }
func (its qwIndexItems) Len() int            { return len(its) }

type qwGroupItem struct {
	group  nip29.Group
	name   string
	search string
}

func newGroupItem(group nip29.Group) qwGroupItem {
	item := qwGroupItem{
		group: group,
	}

	if group.Name != "" {
		item.name = group.Name
		// } else if len(ch.DMRecipients) == 1 {
		// 	item.name = ch.DMRecipients[0].Tag()
		// } else {
		// 	item.name = gtkcord.RecipientNames(ch)
	}

	// if threadTypes[ch.Type] {
	// 	parent, _ := state.Cabinet.Group(ch.ParentID)
	// 	if parent != nil {
	// 		item.name = parent.Name + " › #" + item.name
	// 	}
	// }

	return item
}

func (it qwGroupItem) String() string { return it.search }

const (
	chHash       = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="x-small" rise="-2000">  </span></b></span>`
	chNSFWHash   = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="x-small" rise="-2000">! </span></b></span>`
	chVoiceHash  = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="xx-small" rise="-2000">🔊</span></b></span>`
	chThreadHash = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="x-small" rise="-2000"># </span></b></span>`
)

func (it qwGroupItem) Row(ctx context.Context) *gtk.ListBoxRow {
	tooltip := it.name
	tooltip += " (" + it.group.Name + ")"

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)

	row := gtk.NewListBoxRow()
	row.SetTooltipText(tooltip)
	row.SetChild(box)

	icon := gtk.NewLabel("")
	icon.SetHAlign(gtk.AlignCenter)
	icon.SetMarkup(chHash)
	box.Append(icon)

	name := gtk.NewLabel(it.name)
	name.SetHExpand(true)
	name.SetXAlign(0)
	name.SetEllipsize(pango.EllipsizeEnd)

	box.Append(name)

	relayName := gtk.NewLabel(it.group.Address.Relay)
	relayName.SetEllipsize(pango.EllipsizeEnd)

	box.Append(relayName)

	return row
}

type qwRelayItem struct {
	url string
}

func newRelayItem(url string) qwRelayItem {
	return qwRelayItem{
		url: url,
	}
}

func (it qwRelayItem) String() string { return it.url }

func (it qwRelayItem) Row(ctx context.Context) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()

	// icon := avatar.New(ctx, imgutil.HTTPProvider, gtkcord.InlineEmojiSize)
	// icon.SetInitials(it.Name)
	// icon.SetFromURL(it.IconURL())
	// icon.SetHAlign(gtk.AlignCenter)

	// anim := icon.EnableAnimation()
	// anim.ConnectMotion(row)

	// name := gtk.NewLabel(it.Name)
	// name.SetHExpand(true)
	// name.SetXAlign(0)

	// box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	// box.Append(icon)
	// box.Append(name)

	// row.SetChild(box)
	return row
}

type qwIndex struct {
	items  qwIndexItems
	buffer qwIndexItems
}

const qwSearchLimit = 25

func (idx *qwIndex) update(ctx context.Context) {
	// state := gtkcord.FromContext(ctx).Offline()
	items := make([]qwIndexItem, 0, 250)

	// dms, err := state.PrivateGroups()
	// if err != nil {
	// 	app.Error(ctx, err)
	// 	return
	// }

	// for i := range dms {
	// 	items = append(items, newGroupItem(state, nil, &dms[i]))
	// }

	// relays, err := state.Relays()
	// if err != nil {
	// 	app.Error(ctx, err)
	// 	return
	// }

	// for i, relay := range relays {
	// 	chs, err := state.Groups(relay.ID, gtkcord.AllowedGroupTypes)
	// 	if err != nil {
	// 		log.Print("quickswitcher: cannot populate groups for relay ", relay.Name, ": ", err)
	// 		continue
	// 	}
	// 	items = append(items, newRelayItem(&relays[i]))
	// 	for j := range chs {
	// 		items = append(items, newGroupItem(state, &relays[i], &chs[j]))
	// 	}
	// }

	idx.items = items
}

func (idx *qwIndex) search(str string) []qwIndexItem {
	if idx.items == nil {
		return nil
	}

	idx.buffer = idx.buffer[:0]
	if idx.buffer == nil {
		idx.buffer = make([]qwIndexItem, 0, qwSearchLimit)
	}

	matches := fuzzy.FindFrom(str, idx.items)
	for i := 0; i < len(matches) && i < qwSearchLimit; i++ {
		idx.buffer = append(idx.buffer, idx.items[matches[i].Index])
	}

	return idx.buffer
}
