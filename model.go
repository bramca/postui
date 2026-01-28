package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Focus int
type Tab int

const (
	FocusInput Focus = iota
	FocusResponseView
)

const (
	TabRequestHeaders Tab = iota
	TabRequestBody
	TabResponseBody
	TabResponseHeaders
)

const (
	placeHolderUrl    = "http://localhost:5000/checkout"
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
	spinnerStyle      = lipgloss.NewStyle().Foreground(highlightColor)
)

type keymap = struct {
	nextView, prevView, nextTab, prevTab, left, right, up, down, run, quit key.Binding
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
	activeTab          Tab
	spinner            spinner.Model
	startSpinner       bool
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case responseMsg:
		m.startSpinner = false
		m.err = nil
		m.currentFocus = FocusResponseView
		m.statusCode = msg.statusCode
		m.activeTab = TabResponseBody
		m.responseBody = msg.responseBody
		m.responseHeaders = msg.responseHeaders
		if len(m.tabContent) > 0 {
			m.tabContent[TabResponseBody] = m.responseBody
			m.tabContent[TabResponseHeaders] = m.responseHeaders
			m.responseViewport.SetContent(m.tabContent[m.activeTab])
		}

		for i := range m.inputs {
			// Remove focus from inputs
			m.inputs[i].Blur()
		}
	case errMsg:
		m.startSpinner = false
		m.statusCode = 0
		m.responseBody = ""
		m.err = msg
		m.responseViewport.SetContent(m.tabContent[m.activeTab])

	case spinner.TickMsg:
		if m.startSpinner {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.WindowSizeMsg:
		m.responseViewWidth = msg.Width
		m.responseViewHeight = msg.Height - paddingHeight

		windowStyle = windowStyle.Width(m.responseViewWidth).Height(m.responseViewHeight)

		for i := range m.inputs {
			m.inputs[i].Width = m.responseViewWidth - 20
		}

		m.responseViewport.Width = m.responseViewWidth
		m.responseViewport.Height = m.responseViewHeight

		m.requestHeaders.SetWidth(m.responseViewWidth - 2)
		m.requestHeaders.SetHeight(m.responseViewHeight)

		m.requestBody.SetWidth(m.responseViewWidth - 2)
		m.requestBody.SetHeight(m.responseViewHeight)

		m.updateFocusView()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.left):
			if m.cursorPos-1 >= 0 {
				m.cursorPos -= 1
				m.inputs[m.focusInputIndex].SetCursor(m.cursorPos)
				m.requestHeaders.SetCursor(m.cursorPos)
				m.requestBody.SetCursor(m.cursorPos)
			}
		case key.Matches(msg, m.keymap.right):
			switch m.currentFocus {
			case FocusInput:
				if m.cursorPos+1 <= len(m.inputs[m.focusInputIndex].Value()) {
					m.cursorPos += 1
					m.inputs[m.focusInputIndex].SetCursor(m.cursorPos)
				}
			case FocusResponseView:
				switch m.activeTab {
				case TabRequestHeaders:
					if m.cursorPos+1 <= m.requestHeaders.LineInfo().CharWidth-1 {
						m.cursorPos += 1
						m.requestHeaders.SetCursor(m.cursorPos)
					}
				case TabRequestBody:
					if m.cursorPos+1 <= m.requestBody.LineInfo().CharWidth-1 {
						m.cursorPos += 1
						m.requestBody.SetCursor(m.cursorPos)
					}
				}
			}
		case key.Matches(msg, m.keymap.up):
			switch m.activeTab {
			case TabRequestHeaders:
				m.requestHeaders.CursorUp()
			case TabRequestBody:
				m.requestBody.CursorUp()
			}
		case key.Matches(msg, m.keymap.down):
			switch m.activeTab {
			case TabRequestHeaders:
				m.requestHeaders.CursorDown()
			case TabRequestBody:
				m.requestBody.CursorDown()
			}
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

				for i := 0; i <= len(m.inputs)-1; i++ {
					if i == m.focusInputIndex {
						// Set focused state
						cmds = append(cmds, m.inputs[i].Focus())
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
			case FocusResponseView:
				if s == "alt+[" {
					m.activeTab--
				}
				if s == "alt+]" {
					m.activeTab++
				}

				if int(m.activeTab) >= len(m.tabs) {
					m.activeTab = TabRequestHeaders
				} else if m.activeTab < 0 {
					m.activeTab = Tab(len(m.tabs) - 1)
				}

				switch m.activeTab {
				case TabRequestHeaders:
					m.requestHeaders.Focus()
					m.requestBody.Blur()
				case TabRequestBody:
					m.requestHeaders.Blur()
					m.requestBody.Focus()
				default:
					m.requestHeaders.Blur()
					m.requestBody.Blur()
					m.responseViewport.SetContent(m.tabContent[m.activeTab])
				}

			}
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.run):
			m.startSpinner = true
			url := m.inputs[0].Value()
			method := m.inputs[1].Value()
			headers := map[string]string{}
			body := m.requestBody.Value()
			for line := range strings.SplitSeq(m.requestHeaders.Value(), "\n") {
				lineSplit := strings.Split(line, ":")
				if len(lineSplit) > 1 {
					headers[lineSplit[0]] = lineSplit[1]
				}
			}
			cmds = append(cmds, m.spinner.Tick)
			cmds = append(cmds, doRequest(url, method, headers, body))
		case key.Matches(msg, m.keymap.nextView):
			switch m.currentFocus {
			case FocusInput:
				m.currentFocus = FocusResponseView
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				if m.activeTab == TabRequestHeaders {
					m.requestHeaders.Focus()
					m.cursorPos = m.requestHeaders.LineInfo().CharWidth - 2
				}

				if m.activeTab == TabRequestBody {
					m.requestBody.Focus()
					m.cursorPos = m.requestBody.LineInfo().CharWidth - 2
				}
			case FocusResponseView:
				m.currentFocus = FocusInput
				m.cursorPos = len(m.inputs[m.focusInputIndex].Value())
				m.inputs[m.focusInputIndex].Focus()
				m.requestHeaders.Blur()
				m.requestBody.Blur()
			}
		case key.Matches(msg, m.keymap.prevView):
			switch m.currentFocus {
			case FocusInput:
				m.currentFocus = FocusResponseView
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				if m.activeTab == TabRequestHeaders {
					m.requestHeaders.Focus()
				}

				if m.activeTab == TabRequestBody {
					m.requestBody.Focus()
				}
			case FocusResponseView:
				m.currentFocus = FocusInput
				m.cursorPos = len(m.inputs[m.focusInputIndex].Value())
				m.inputs[m.focusInputIndex].Focus()
				m.requestHeaders.Blur()
				m.requestBody.Blur()
			}
		default:
			if len(msg.String()) == 1 || msg.String() == "backspace" || msg.String() == "enter" {
				cmd := m.updateInputs(msg)
				cmds = append(cmds, cmd)
				m.cursorPos += 1
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder
	var renderedTabs []string

	m.updateFocusView()

	tabWidth := m.responseViewWidth / len(m.tabs)
	for i, t := range m.tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.tabs)-1, i == int(m.activeTab)
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
		if m.startSpinner && i == 0 {
			b.WriteString("    " + m.spinner.View())
		}
		b.WriteRune('\n')
	}

	b.WriteString(row)
	b.WriteRune('\n')
	switch m.activeTab {
	case TabRequestHeaders:
		b.WriteString(m.requestHeaders.View())
	case TabRequestBody:
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

func (m *model) updateFocusView() {
	switch m.currentFocus {
	case FocusResponseView:
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
}

func initialModel() model {
	m := model{
		help:         help.New(),
		inputs:       make([]textinput.Model, 2),
		tabs:         []string{"Request Headers", "Request Body", "Response Body", "Response Headers"},
		currentFocus: FocusInput,
		spinner:      spinner.New(),
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
			up: key.NewBinding(
				key.WithKeys("up"),
				key.WithHelp("up", "move cursor up"),
			),
			down: key.NewBinding(
				key.WithKeys("down"),
				key.WithHelp("down", "move cursor down"),
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

	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinner.Moon

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
