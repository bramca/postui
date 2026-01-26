package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const padding = 2

var (
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	focusedStyle      = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle       = focusedStyle
	noStyle           = lipgloss.NewStyle()
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	nonHighlightColor = lipgloss.Color("#838383")
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor).Padding(0, padding)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Padding(padding, 0).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
)

type keymap = struct {
	nextInput, prevInput, nextTab, prevTab, focusResponse, focusInput, run, quit key.Binding
}

type model struct {
	windowWidth      int
	windowHeight     int
	focusInputIndex  int
	keymap           keymap
	help             help.Model
	cursorMode       cursor.Mode
	inputs           []textinput.Model
	statusCode       int
	responseBody     string
	responseHeaders  string
	err              error
	responseViewport viewport.Model
	tabs             []string
	tabContent       []string
	activeTab        int
	focusInputs      bool
	focusResponse    bool
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.focusResponse {
		m.responseViewport, cmd = m.responseViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case responseMsg:
		var err error
		m.err = nil
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

		m.focusInputs = false
		m.focusResponse = true

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
		windowStyle = windowStyle.Width(msg.Width - padding).Height(msg.Height - 20)
		m.windowWidth = msg.Width - padding
		m.windowHeight = msg.Height - 20
		m.responseViewport.Width = msg.Width - padding
		m.responseViewport.Height = msg.Height - 20

		return m, nil
	case tea.KeyMsg:
		switch {
		// Set focus to next input
		case key.Matches(msg, m.keymap.nextInput), key.Matches(msg, m.keymap.prevInput):
			s := msg.String()
			m.focusInputs = true
			m.focusResponse = false

			// Cycle indexes
			if s == "shift+tab" {
				m.focusInputIndex--
			}
			if s == "tab" {
				m.focusInputIndex++
			}

			if m.focusInputIndex >= len(m.inputs) {
				m.focusInputIndex = 0
			} else if m.focusInputIndex < 0 {
				m.focusInputIndex = len(m.inputs)
			}

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

			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.run):
			return m, doRequest(m.inputs[0].Value())
		case key.Matches(msg, m.keymap.nextTab):
			var err error
			m.activeTab = min(m.activeTab+1, len(m.tabs)-1)
			m.responseViewport, err = setResponseViewportContent(m.responseViewport, m.tabContent[m.activeTab])

			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}
		case key.Matches(msg, m.keymap.prevTab):
			var err error
			m.activeTab = max(m.activeTab-1, 0)
			m.responseViewport, err = setResponseViewportContent(m.responseViewport, m.tabContent[m.activeTab])

			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}
		case key.Matches(msg, m.keymap.focusInput):
			m.inputs[m.focusInputIndex].Focus()
			m.focusInputs = true
			m.focusResponse = false

			return m, nil
		case key.Matches(msg, m.keymap.focusResponse):
			m.focusResponse = true
			m.focusInputs = false
			for i := range m.inputs {
				m.inputs[i].Blur()
			}

			return m, nil
		default:
			if len(msg.String()) == 1 || msg.String() == "backspace" {
				cmd = m.updateInputs(msg)
			}
		}
	}

	return m, cmd
}

func (m model) View() string {
	var b strings.Builder
	var renderedTabs []string

	if m.focusResponse {
		for i := range m.inputs {
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
		windowStyle = windowStyle.BorderForeground(highlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(highlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseViewport.Style = windowStyle
	}
	if m.focusInputs {
		m.inputs[m.focusInputIndex].PromptStyle = focusedStyle
		m.inputs[m.focusInputIndex].TextStyle = focusedStyle
		windowStyle = windowStyle.BorderForeground(nonHighlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(nonHighlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseViewport.Style = windowStyle
	}

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
		style = style.Width(m.windowWidth/len(m.tabs) - padding).Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
	}

	b.WriteRune('\n')
	fmt.Fprintf(&b, "focus: %d", m.focusInputIndex)
	b.WriteRune('\n')

	b.WriteString(row)
	b.WriteRune('\n')
	b.WriteString(m.responseViewport.View())
	b.WriteRune('\n')

	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.nextInput,
		m.keymap.prevInput,
		m.keymap.nextTab,
		m.keymap.prevTab,
		m.keymap.focusInput,
		m.keymap.focusResponse,
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
		help:        help.New(),
		inputs:      make([]textinput.Model, 2),
		tabs:        []string{"Response Body", "Response Headers"},
		focusInputs: true,
		keymap: keymap{
			nextInput: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next input"),
			),
			prevInput: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "prev input"),
			),
			nextTab: key.NewBinding(
				key.WithKeys("right"),
				key.WithHelp("right", "next tab"),
			),
			prevTab: key.NewBinding(
				key.WithKeys("left"),
				key.WithHelp("left", "prev tab"),
			),
			focusResponse: key.NewBinding(
				key.WithKeys("ctrl+e"),
				key.WithHelp("ctrl+e", "focus response"),
			),
			focusInput: key.NewBinding(
				key.WithKeys("ctrl+i"),
				key.WithHelp("ctrl+i", "focus inputs"),
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
