package postui

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	help                 key.Binding
	nextView             key.Binding
	prevView             key.Binding
	nextTab              key.Binding
	prevTab              key.Binding
	left                 key.Binding
	right                key.Binding
	up                   key.Binding
	down                 key.Binding
	scrollUpMulti        key.Binding
	scrollDownMulti      key.Binding
	j                    key.Binding
	k                    key.Binding
	l                    key.Binding
	h                    key.Binding
	top                  key.Binding
	bottom               key.Binding
	paste                key.Binding
	copy                 key.Binding
	copyCurl             key.Binding
	save                 key.Binding
	run                  key.Binding
	addCollection        key.Binding
	extractCollection    key.Binding
	toggleCollectionEdit key.Binding
	collectionListSelect key.Binding
	extractHeaders       key.Binding
	quit                 key.Binding
}

func NewKeymap() keymap {
	return keymap{
		help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "help"),
		),
		nextView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next view"),
		),
		prevView: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev view"),
		),
		nextTab: key.NewBinding(
			key.WithKeys("alt+]"),
			key.WithHelp("alt+]", "next tab"),
		),
		prevTab: key.NewBinding(
			key.WithKeys("alt+["),
			key.WithHelp("alt+[", "prev tab"),
		),
		left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("left", "move cursor left"),
		),
		right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("right", "move cursor right"),
		),
		up: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "move cursor up"),
		),
		down: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("ctrl+j", "move cursor down"),
		),
		scrollUpMulti: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "scroll up multi"),
		),
		scrollDownMulti: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "scroll down multi"),
		),
		toggleCollectionEdit: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "toggle edit mode"),
		),
		collectionListSelect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select list item"),
		),
		h: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "scroll left"),
		),
		j: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "scroll down"),
		),
		k: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "scroll up"),
		),
		l: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "scroll right"),
		),
		top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "scroll top"),
		),
		bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "scroll bottom"),
		),
		paste: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("ctrl+v", "paste"),
		),
		copy: key.NewBinding(
			key.WithKeys("alt+x"),
			key.WithHelp("alt+x", "copy"),
		),
		copyCurl: key.NewBinding(
			key.WithKeys("alt+c"),
			key.WithHelp("alt+c", "copy curl"),
		),
		save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		run: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "run"),
		),
		addCollection: key.NewBinding(
			key.WithKeys("alt+a"),
			key.WithHelp("alt+a", "collection add"),
		),
		extractCollection: key.NewBinding(
			key.WithKeys("alt+e"),
			key.WithHelp("alt+e", "collection extract"),
		),
		extractHeaders: key.NewBinding(
			key.WithKeys("alt+h"),
			key.WithHelp("alt+h", "headers extract"),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.help,
		k.nextView,
		k.prevView,
		k.nextTab,
		k.prevTab,
		k.run,
		k.addCollection,
		k.extractCollection,
		k.copy,
		k.save,
		k.quit,
	}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.help,
			k.nextView,
			k.prevView,
			k.nextTab,
			k.prevTab,
			k.run,
		},
		{
			k.addCollection,
			k.extractCollection,
			k.extractHeaders,
			k.copy,
			k.paste,
			k.save,
			k.copyCurl,
			k.quit,
		},
		{
			k.left,
			k.right,
			k.up,
			k.down,
			k.scrollUpMulti,
			k.scrollDownMulti,
			k.toggleCollectionEdit,
		},
		{
			k.collectionListSelect,
			k.h,
			k.j,
			k.k,
			k.l,
			k.top,
			k.bottom,
		},
	}
}
