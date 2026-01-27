package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Focus int

const (
	FocusInput Focus = iota
	FocusResponse
	FocusRequestHeaders
	FocusRequestBody
)

const (
	placeHolderUrl    = "https://v2.jokeapi.dev/joke/Any"
	placeHolderMethod = "GET"
	paddingHeight     = 8
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
	responseViewWidth  int
	responseViewHeight int
	focusInputIndex    int
	currentFocus       Focus
	cursorPos          int
	keymap             keymap
	help               help.Model
	inputs             []textinput.Model
	statusCode         int
	responseBody       string
	responseHeaders    string
	err                error
	responseViewport   viewport.Model
	requestHeaders     textarea.Model
	requestBody        textarea.Model
	tabs               []string
	tabContent         []string
	activeTab          int
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.currentFocus == FocusResponse {
		m.responseViewport, cmd = m.responseViewport.Update(msg)
		m.tabContent[0] = m.requestHeaders.Value()
		m.tabContent[1] = m.requestBody.Value()
	}

	switch msg := msg.(type) {
	case responseMsg:
		m.err = nil
		m.currentFocus = FocusResponse
		m.statusCode = msg.statusCode
		m.activeTab = 2
		m.responseBody = msg.responseBody
		m.responseHeaders = msg.responseHeaders
		if len(m.tabContent) > 0 {
			m.tabContent[2] = m.responseBody
			m.tabContent[3] = m.responseHeaders
			m.responseViewport.SetContent(m.tabContent[m.activeTab])
		}

		for i := range m.inputs {
			// Remove focus from inputs
			m.inputs[i].Blur()
		}

		return m, nil
	case errMsg:
		m.statusCode = 0
		m.responseBody = ""
		m.err = msg
		m.responseViewport.SetContent(m.tabContent[m.activeTab])

		return m, nil
	case tea.WindowSizeMsg:
		m.responseViewWidth = msg.Width
		m.responseViewHeight = msg.Height - paddingHeight

		windowStyle = windowStyle.Width(m.responseViewWidth).Height(m.responseViewHeight)

		m.responseViewport.Width = m.responseViewWidth
		m.responseViewport.Height = m.responseViewHeight

		m.requestHeaders.SetWidth(m.responseViewWidth - 2)
		m.requestHeaders.SetHeight(m.responseViewHeight)

		m.requestBody.SetWidth(m.responseViewWidth - 2)
		m.requestBody.SetHeight(m.responseViewHeight)

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
						m.requestHeaders.Blur()
						m.requestBody.Blur()
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

				switch m.activeTab {
				case 0:
					m.requestHeaders.Focus()
					m.requestBody.Blur()
				case 1:
					m.requestHeaders.Blur()
					m.requestBody.Focus()
				default:
					m.requestHeaders.Blur()
					m.requestBody.Blur()
					m.responseViewport.SetContent(m.tabContent[m.activeTab])
				}

				return m, nil

			}
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.run):
			return m, doRequest(m.inputs[0].Value(), m.inputs[1].Value(), "")
		case key.Matches(msg, m.keymap.nextView):
			switch m.currentFocus {
			case FocusInput:
				m.currentFocus = FocusResponse
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				if m.activeTab == 0 {
					m.requestHeaders.Focus()
				}

				if m.activeTab == 1 {
					m.requestBody.Focus()
				}
			case FocusResponse:
				m.currentFocus = FocusInput
				m.cursorPos = len(m.inputs[m.focusInputIndex].Value())
				m.inputs[m.focusInputIndex].Focus()
				m.requestHeaders.Blur()
				m.requestBody.Blur()
			}
		case key.Matches(msg, m.keymap.prevView):
			switch m.currentFocus {
			case FocusInput:
				m.currentFocus = FocusResponse
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				if m.activeTab == 0 {
					m.requestHeaders.Focus()
				}

				if m.activeTab == 1 {
					m.requestBody.Focus()
				}
			case FocusResponse:
				m.currentFocus = FocusInput
				m.cursorPos = len(m.inputs[m.focusInputIndex].Value())
				m.inputs[m.focusInputIndex].Focus()
				m.requestHeaders.Blur()
				m.requestBody.Blur()
			}
		default:
			if len(msg.String()) == 1 || msg.String() == "backspace" || msg.String() == "enter" {
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

	tabWidth := m.responseViewWidth / len(m.tabs)
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
			tabWidth = (m.responseViewWidth / len(m.tabs)) + m.responseViewWidth%len(m.tabs) - 2*len(m.tabs)
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
	switch m.activeTab {
	case 0:
		b.WriteString(m.requestHeaders.View())
	case 1:
		b.WriteString(m.requestBody.View())
	default:
		b.WriteString(m.responseViewport.View())
	}
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
	cmds := make([]tea.Cmd, len(m.inputs)+1)

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	var i int
	for i = range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	m.requestHeaders, cmds[i] = m.requestHeaders.Update(msg)
	m.requestBody, cmds[i+1] = m.requestBody.Update(msg)

	return tea.Batch(cmds...)
}

func initialModel() model {
	m := model{
		help:         help.New(),
		inputs:       make([]textinput.Model, 2),
		tabs:         []string{"Request Headers", "Request Body", "Response Body", "Response Headers"},
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
			t.CharLimit = 10
			t.SetValue(placeHolderMethod)
			t.Width = t.CharLimit
		}

		m.inputs[i] = t
	}

	m.responseViewport = viewport.New(78, 20)
	m.responseViewport.Style = windowStyle

	m.requestHeaders = textarea.New()
	m.requestHeaders.Cursor.Style = cursorStyle
	m.requestHeaders.BlurredStyle.Base = windowStyle
	m.requestHeaders.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestBody = textarea.New()
	m.requestBody.Cursor.Style = cursorStyle
	m.requestBody.BlurredStyle.Base = windowStyle
	m.requestBody.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	return m
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right

	return border
}
