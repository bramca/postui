package postui

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	help                 key.Binding
	nextView             key.Binding
	prevView             key.Binding
	nextTab              key.Binding
	prevTab              key.Binding
	up                   key.Binding
	down                 key.Binding
	left                 key.Binding
	right                key.Binding
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
	focusCollection      key.Binding
	toggleCollectionEdit key.Binding
	quit                 key.Binding
}

func NewKeymap() keymap {
	return keymap{
		help: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "help"),
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
		up: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "move cursor up"),
		),
		down: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("ctrl+j", "move cursor down"),
		),
		left: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("cltr+h", "move cursor left"),
		),
		right: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("cltr+l", "move cursor right"),
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
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle edit mode"),
		),
		h: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "go back/scroll left"),
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
			key.WithHelp("l", "select/scroll right"),
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
			key.WithKeys("alt+c"),
			key.WithHelp("alt+c", "copy"),
		),
		copyCurl: key.NewBinding(
			key.WithKeys("alt+x"),
			key.WithHelp("alt+x", "copy curl"),
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
		focusCollection: key.NewBinding(
			key.WithKeys("alt+l"),
			key.WithHelp("alt+l", "collection focus"),
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
		k.h,
		k.j,
		k.k,
		k.l,
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
			k.copy,
			k.paste,
			k.save,
			k.copyCurl,
			k.quit,
		},
		{
			k.up,
			k.down,
			k.left,
			k.right,
		},
		{
			k.scrollUpMulti,
			k.scrollDownMulti,
			k.top,
			k.bottom,
		},
		{
			k.toggleCollectionEdit,
			k.focusCollection,
			k.h,
			k.j,
			k.k,
			k.l,
		},
	}
}
