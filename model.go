package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Focus int

const (
	FocusInput Focus = iota
	FocusResponse
)

const (
	paddingHeight = 10
)

var (
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	focusedStyle      = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle       = focusedStyle
	noStyle           = lipgloss.NewStyle()
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	nonHighlightColor = lipgloss.Color("#838383")
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
)

type keymap = struct {
	nextView, prevView, nextTab, prevTab, left, right, run, quit key.Binding
}

type model struct {
	windowWidth      int
	windowHeight     int
	focusInputIndex  int
	currentFocus     Focus
	cursorPos        int
	keymap           keymap
	help             help.Model
	inputs           []textinput.Model
	statusCode       int
	responseBody     string
	responseHeaders  string
	err              error
	responseViewport viewport.Model
	tabs             []string
	tabContent       []string
	activeTab        int
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.currentFocus == FocusResponse {
		m.responseViewport, cmd = m.responseViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case responseMsg:
		var err error
		m.err = nil
		m.currentFocus = FocusResponse
		m.statusCode = msg.statusCode
		m.responseBody = msg.responseBody
		m.responseHeaders = msg.responseHeaders
		if len(m.tabContent) > 0 {
			m.tabContent[0] = m.responseBody
			m.tabContent[1] = m.responseHeaders
			m.responseViewport, err = setResponseViewportContent(m.responseViewport, m.tabContent[m.activeTab])
		}

		for i := range m.inputs {
			// Remove focus from inputs
			m.inputs[i].Blur()
		}

		if err != nil {
			return m, func() tea.Msg {
				return errMsg{err: err}
			}
		}

		return m, nil
	case errMsg:
		m.statusCode = 0
		m.responseBody = ""
		m.err = msg
		m.responseViewport, _ = setResponseViewportContent(m.responseViewport, m.err.Error())

		return m, nil
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height - paddingHeight
		windowStyle = windowStyle.Width(m.windowWidth).Height(m.windowHeight)
		m.responseViewport.Width = m.windowWidth
		m.responseViewport.Height = m.windowHeight

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.left):
			if m.cursorPos-1 >= 0 {
				m.cursorPos -= 1
				m.inputs[m.focusInputIndex].SetCursor(m.cursorPos)
			}
		case key.Matches(msg, m.keymap.right):
			if m.cursorPos+1 <= len(m.inputs[m.focusInputIndex].Value()) {
				m.cursorPos += 1
				m.inputs[m.focusInputIndex].SetCursor(m.cursorPos)
			}
		// Set focus to next input
		case key.Matches(msg, m.keymap.nextTab), key.Matches(msg, m.keymap.prevTab):
			s := msg.String()
			switch m.currentFocus {
			case FocusInput:
				if s == "alt+[" {
					m.focusInputIndex--
				}
				if s == "alt+]" {
					m.focusInputIndex++
				}

				if m.focusInputIndex >= len(m.inputs) {
					m.focusInputIndex = 0
				} else if m.focusInputIndex < 0 {
					m.focusInputIndex = len(m.inputs) - 1
				}

				m.cursorPos = len(m.inputs[m.focusInputIndex].Value())

				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i <= len(m.inputs)-1; i++ {
					if i == m.focusInputIndex {
						// Set focused state
						cmds[i] = m.inputs[i].Focus()
						m.inputs[i].PromptStyle = focusedStyle
						m.inputs[i].TextStyle = focusedStyle
						continue
					}
					// Remove focused state
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = noStyle
					m.inputs[i].TextStyle = noStyle
				}

				// Cycle indexes

				return m, tea.Batch(cmds...)
			case FocusResponse:
				var err error
				if s == "alt+[" {
					m.activeTab--
				}
				if s == "alt+]" {
					m.activeTab++
				}

				if m.activeTab >= len(m.tabs) {
					m.activeTab = 0
				} else if m.activeTab < 0 {
					m.activeTab = len(m.tabs) - 1
				}

				m.responseViewport, err = setResponseViewportContent(m.responseViewport, m.tabContent[m.activeTab])
				if err != nil {
					return m, func() tea.Msg {
						return errMsg{err: err}
					}
				}

				return m, nil

			}
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.run):
			return m, doRequest(m.inputs[0].Value())
		case key.Matches(msg, m.keymap.nextView):
			switch m.currentFocus {
			case FocusInput:
				m.currentFocus = FocusResponse
			case FocusResponse:
				m.currentFocus = FocusInput
			}
		case key.Matches(msg, m.keymap.prevView):
			switch m.currentFocus {
			case FocusInput:
				m.currentFocus = FocusResponse
			case FocusResponse:
				m.currentFocus = FocusInput
			}
		default:
			if len(msg.String()) == 1 || msg.String() == "backspace" {
				cmd = m.updateInputs(msg)
				m.cursorPos += 1
			}
		}
	}

	return m, cmd
}

func (m model) View() string {
	var b strings.Builder
	var renderedTabs []string

	switch m.currentFocus {
	case FocusResponse:
		for i := range m.inputs {
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
		windowStyle = windowStyle.BorderForeground(highlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(highlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseViewport.Style = windowStyle
	case FocusInput:
		m.inputs[m.focusInputIndex].PromptStyle = focusedStyle
		m.inputs[m.focusInputIndex].TextStyle = focusedStyle
		windowStyle = windowStyle.BorderForeground(nonHighlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(nonHighlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseViewport.Style = windowStyle
	}

	tabWidth := m.windowWidth / len(m.tabs)
	for i, t := range m.tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		if i == len(m.tabs)-1 {
			tabWidth = (m.windowWidth / len(m.tabs)) + m.windowWidth%len(m.tabs) - 2*len(m.tabs)
		}
		style = style.Width(tabWidth).Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
	}

	b.WriteString(row)
	b.WriteRune('\n')
	b.WriteString(m.responseViewport.View())
	b.WriteRune('\n')

	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.nextView,
		m.keymap.prevView,
		m.keymap.nextTab,
		m.keymap.prevTab,
		m.keymap.run,
		m.keymap.quit,
	})

	b.WriteString(help)

	return b.String()
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func initialModel() model {
	m := model{
		help:         help.New(),
		inputs:       make([]textinput.Model, 2),
		tabs:         []string{"Response Body", "Response Headers"},
		currentFocus: FocusInput,
		keymap: keymap{
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
			run: key.NewBinding(
				key.WithKeys("ctrl+r"),
				key.WithHelp("ctrl+r", "run"),
			),
			quit: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "quit"),
			),
		},
	}

	m.tabContent = make([]string, len(m.tabs))

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle

		switch i {
		// URL
		case 0:
			t.CharLimit = 256
			t.SetValue(placeHolderUrl)
			m.cursorPos = len(placeHolderUrl)
			t.Width = t.CharLimit
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		// Headers
		case 1:
			t.CharLimit = 100
			t.Placeholder = "Authorization: "
			t.Width = t.CharLimit
		}

		m.inputs[i] = t
	}

	m.responseViewport = viewport.New(78, 20)
	m.responseViewport.Style = windowStyle

	return m
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right

	return border
}

func setResponseViewportContent(vp viewport.Model, content string) (viewport.Model, error) {
	vp.SetContent(content)

	return vp, nil
}
