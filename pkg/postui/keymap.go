package postui

import "github.com/charmbracelet/bubbles/key"

type Keymap struct {
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
	goDown               key.Binding
	goUp                 key.Binding
	goForward            key.Binding
	goBack               key.Binding
	top                  key.Binding
	bottom               key.Binding
	paste                key.Binding
	copy                 key.Binding
	copyCurl             key.Binding
	save                 key.Binding
	run                  key.Binding
	addCollection        key.Binding
	focusCollection      key.Binding
	focusResponse        key.Binding
	toggleCollectionEdit key.Binding
	reloadConfig         key.Binding
	search               key.Binding
	searchNext           key.Binding
	searchPrev           key.Binding
	searchStop           key.Binding
	quit                 key.Binding
	captureResult        key.Binding
	pasteResult          key.Binding
}

func NewKeymapWithBindings(bindings Keybindings) Keymap {
	return Keymap{
		help: key.NewBinding(
			key.WithKeys(bindings.Help),
			key.WithHelp(bindings.Help, "help"),
		),
		nextView: key.NewBinding(
			key.WithKeys(bindings.NextView),
			key.WithHelp(bindings.NextView, "next view"),
		),
		prevView: key.NewBinding(
			key.WithKeys(bindings.PrevView),
			key.WithHelp(bindings.PrevView, "prev view"),
		),
		nextTab: key.NewBinding(
			key.WithKeys(bindings.NextTab),
			key.WithHelp(bindings.NextTab, "next tab"),
		),
		prevTab: key.NewBinding(
			key.WithKeys(bindings.PrevTab),
			key.WithHelp(bindings.PrevTab, "prev tab"),
		),
		up: key.NewBinding(
			key.WithKeys(bindings.Up),
			key.WithHelp(bindings.Up, "move cursor up"),
		),
		down: key.NewBinding(
			key.WithKeys(bindings.Down),
			key.WithHelp(bindings.Down, "move cursor down"),
		),
		left: key.NewBinding(
			key.WithKeys(bindings.Left),
			key.WithHelp(bindings.Left, "move cursor left"),
		),
		right: key.NewBinding(
			key.WithKeys(bindings.Right),
			key.WithHelp(bindings.Right, "move cursor right"),
		),
		scrollUpMulti: key.NewBinding(
			key.WithKeys(bindings.ScrollUpMulti),
			key.WithHelp(bindings.ScrollUpMulti, "scroll up multi"),
		),
		scrollDownMulti: key.NewBinding(
			key.WithKeys(bindings.ScrollDownMulti),
			key.WithHelp(bindings.ScrollDownMulti, "scroll down multi"),
		),
		toggleCollectionEdit: key.NewBinding(
			key.WithKeys(bindings.ToggleCollectionEdit),
			key.WithHelp(bindings.ToggleCollectionEdit, "toggle edit mode"),
		),
		goBack: key.NewBinding(
			key.WithKeys(bindings.GoBack),
			key.WithHelp(bindings.GoBack, "go back/scroll left"),
		),
		goDown: key.NewBinding(
			key.WithKeys(bindings.GoDown),
			key.WithHelp(bindings.GoDown, "scroll down"),
		),
		goUp: key.NewBinding(
			key.WithKeys(bindings.GoUp),
			key.WithHelp(bindings.GoUp, "scroll up"),
		),
		goForward: key.NewBinding(
			key.WithKeys(bindings.GoForward),
			key.WithHelp(bindings.GoForward, "select/scroll right"),
		),
		top: key.NewBinding(
			key.WithKeys(bindings.Top),
			key.WithHelp(bindings.Top, "scroll top"),
		),
		bottom: key.NewBinding(
			key.WithKeys(bindings.Bottom),
			key.WithHelp(bindings.Bottom, "scroll bottom"),
		),
		paste: key.NewBinding(
			key.WithKeys(bindings.Paste),
			key.WithHelp(bindings.Paste, "paste"),
		),
		copy: key.NewBinding(
			key.WithKeys(bindings.Copy),
			key.WithHelp(bindings.Copy, "copy"),
		),
		copyCurl: key.NewBinding(
			key.WithKeys(bindings.CopyCurl),
			key.WithHelp(bindings.CopyCurl, "copy curl"),
		),
		save: key.NewBinding(
			key.WithKeys(bindings.Save),
			key.WithHelp(bindings.Save, "save"),
		),
		run: key.NewBinding(
			key.WithKeys(bindings.Run),
			key.WithHelp(bindings.Run, "run"),
		),
		addCollection: key.NewBinding(
			key.WithKeys(bindings.AddCollection),
			key.WithHelp(bindings.AddCollection, "collection add"),
		),
		focusCollection: key.NewBinding(
			key.WithKeys(bindings.FocusCollection),
			key.WithHelp(bindings.FocusCollection, "collection focus"),
		),
		focusResponse: key.NewBinding(
			key.WithKeys(bindings.FocusResponse),
			key.WithHelp(bindings.FocusResponse, "response focus"),
		),
		reloadConfig: key.NewBinding(
			key.WithKeys(bindings.ReloadConfig),
			key.WithHelp(bindings.ReloadConfig, "reload config"),
		),
		search: key.NewBinding(
			key.WithKeys(bindings.Search),
			key.WithHelp(bindings.Search, "search"),
		),
		searchNext: key.NewBinding(
			key.WithKeys(bindings.SearchNext),
			key.WithHelp(bindings.SearchNext, "next search"),
		),
		searchPrev: key.NewBinding(
			key.WithKeys(bindings.SearchPrev),
			key.WithHelp(bindings.SearchPrev, "previous search"),
		),
		searchStop: key.NewBinding(
			key.WithKeys(bindings.SearchStop),
			key.WithHelp(bindings.SearchStop, "stop search"),
		),
		quit: key.NewBinding(
			key.WithKeys(bindings.Quit),
			key.WithHelp(bindings.Quit, "quit"),
		),
		captureResult: key.NewBinding(
			key.WithKeys(
				"0",
				"1",
				"2",
				"3",
				"4",
				"5",
				"6",
				"7",
				"8",
				"9",
			),
			key.WithHelp("0-9", "capture result"),
		),
		pasteResult: key.NewBinding(
			key.WithKeys(
				"alt+0",
				"alt+1",
				"alt+2",
				"alt+3",
				"alt+4",
				"alt+5",
				"alt+6",
				"alt+7",
				"alt+8",
				"alt+9",
			),
			key.WithHelp("alt+0-9", "paste result"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k Keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.help,
		k.nextView,
		k.prevView,
		k.nextTab,
		k.prevTab,
		k.goForward,
		k.goBack,
		k.run,
		k.search,
		k.quit,
	}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k Keymap) FullHelp() [][]key.Binding {
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
			k.scrollUpMulti,
			k.scrollDownMulti,
		},
		{
			k.top,
			k.bottom,
			k.toggleCollectionEdit,
			k.focusCollection,
			k.focusResponse,
			k.reloadConfig,
		},
		{
			k.goBack,
			k.goDown,
			k.goUp,
			k.goForward,
		},
		{
			k.search,
			k.searchNext,
			k.searchPrev,
			k.searchStop,
			k.captureResult,
			k.pasteResult,
		},
	}
}
