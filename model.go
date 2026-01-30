package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	Focus int
	Tab   int
)

const (
	FocusInput Focus = iota
	FocusResponseView
)

const (
	TabCollection Tab = iota
	TabRequestHeaders
	TabRequestBody
	TabResponseBody
	TabResponseHeaders
)

const (
	placeHolderUrl    = "https://v2.jokeapi.dev/joke/Any?type=twopart"
	placeHolderMethod = "GET"
	paddingHeight     = 8
)

var (
	highlightColor      = lipgloss.AdaptiveColor{Light: "#5A647E", Dark: "#519F50"}
	focusedStyle        = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	inactiveTabBorder   = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder     = tabBorderWithBottom("┘", " ", "└")
	nonHighlightColor   = lipgloss.Color("#535353")
	inactiveTabStyle    = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor)
	activeTabStyle      = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle         = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	spinnerStyle        = lipgloss.NewStyle().Foreground(highlightColor)
	statusCodeViewStyle = lipgloss.NewStyle().Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
)

type keymap = struct {
	nextView, prevView, nextTab, prevTab, left, right, up, down, j, k, l, h, paste, run, addCollection, extractCollection, quit key.Binding
}

type model struct {
	inputs         []textinput.Model
	statusCodeView viewport.Model
	spinner        spinner.Model
	responseView   viewport.Model
	requestHeaders textarea.Model
	requestBody    textarea.Model
	collection     textarea.Model
	help           help.Model

	activeTab    Tab
	currentFocus Focus
	keymap       keymap

	responseViewWidth  int
	responseViewHeight int
	focusInputIndex    int
	cursorPos          int
	statusCode         int
	responseTime       int64
	err                error
	startSpinner       bool
	responseBody       string
	responseHeaders    string
	tabs               []string
	tabContent         []string
	collectionMap      map[string]any

	testExtract string
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
		m.responseTime = msg.responseTime
		if len(m.tabContent) > 0 {
			m.tabContent[TabResponseBody] = m.responseBody
			m.tabContent[TabResponseHeaders] = m.responseHeaders
			m.responseView.SetContent(m.tabContent[m.activeTab])
		}

		if m.statusCode > 0 {
			if m.statusCode < 300 {
				statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
			}

			if m.statusCode > 299 && m.statusCode < 400 {
				statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#FFC66D"})
			}

			if m.statusCode > 399 {
				statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#DA4939"})
			}
			statusMsg := fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode))
			padding := (m.statusCodeView.Width - len(statusMsg)) / 2
			m.statusCodeView.SetContent(fmt.Sprintf("%s%s", strings.Repeat(" ", padding), statusMsg))
			m.statusCodeView.Style = statusCodeViewStyle
		}

		for i := range m.inputs {
			// Remove focus from inputs
			m.inputs[i].Blur()
		}
	case errMsg:
		m.startSpinner = false
		m.statusCode = 0
		m.responseTime = 0
		m.responseBody = ""
		m.err = msg
		m.responseView.SetContent(m.tabContent[m.activeTab])

	case spinner.TickMsg:
		if m.startSpinner {
			var cmd tea.Cmd
			m.statusCode = 0
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

		m.responseView.Width = m.responseViewWidth
		m.responseView.Height = m.responseViewHeight

		m.requestHeaders.SetWidth(m.responseViewWidth)
		m.requestHeaders.SetHeight(m.responseViewHeight)

		m.collection.SetWidth(m.responseViewWidth)
		m.collection.SetHeight(m.responseViewHeight)

		m.requestBody.SetWidth(m.responseViewWidth)
		m.requestBody.SetHeight(m.responseViewHeight)

		m.responseView.Style = windowStyle

		m.updateFocusView()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.left):
			m.updateCursorPos(m.cursorPos - 1)
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollLeft(1)
			}
		case key.Matches(msg, m.keymap.right):
			m.updateCursorPos(m.cursorPos + 1)
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollLeft(1)
			}
		case key.Matches(msg, m.keymap.h):
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollLeft(1)
			}
		case key.Matches(msg, m.keymap.j):
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollDown(1)
			}
		case key.Matches(msg, m.keymap.k):
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollUp(1)
			}
		case key.Matches(msg, m.keymap.l):
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollRight(1)
			}
		case key.Matches(msg, m.keymap.up):
			switch m.activeTab {
			case TabCollection:
				m.collection.CursorUp()
			case TabRequestHeaders:
				m.requestHeaders.CursorUp()
			case TabRequestBody:
				m.requestBody.CursorUp()
			case TabResponseBody, TabResponseHeaders:
				m.responseView.ScrollUp(1)
			}
		case key.Matches(msg, m.keymap.down):
			switch m.activeTab {
			case TabCollection:
				m.collection.CursorDown()
			case TabRequestHeaders:
				m.requestHeaders.CursorDown()
			case TabRequestBody:
				m.requestBody.CursorDown()
			case TabResponseBody, TabResponseHeaders:
				m.responseView.ScrollDown(1)
			}
		case key.Matches(msg, m.keymap.paste):
			cb, err := clipboard.ReadAll()
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}
			switch m.currentFocus {
			case FocusInput:
				currentInput := m.inputs[m.focusInputIndex]
				if m.cursorPos >= len(currentInput.Value())-1 {
					m.inputs[m.focusInputIndex].SetValue(currentInput.Value() + cb)
				} else {
					m.inputs[m.focusInputIndex].SetValue(currentInput.Value()[0:m.cursorPos] + cb + currentInput.Value()[m.cursorPos:len(currentInput.Value())-1])
				}
				m.cursorPos += len(cb)
			case FocusResponseView:
				switch m.activeTab {
				case TabCollection:
					m.collection.InsertString(cb)
				case TabRequestHeaders:
					m.requestHeaders.InsertString(cb)
				case TabRequestBody:
					m.requestBody.InsertString(cb)
				}
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
						m.collection.Blur()
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
					m.activeTab = TabCollection
				} else if m.activeTab < 0 {
					m.activeTab = Tab(len(m.tabs) - 1)
				}

				switch m.activeTab {
				case TabCollection:
					m.collection.Focus()
					m.requestBody.Blur()
					m.requestHeaders.Blur()
				case TabRequestHeaders:
					m.requestHeaders.Focus()
					m.requestBody.Blur()
					m.collection.Blur()
				case TabRequestBody:
					m.requestBody.Focus()
					m.requestHeaders.Blur()
					m.collection.Blur()
				default:
					m.requestHeaders.Blur()
					m.collection.Blur()
					m.requestBody.Blur()
					m.responseView.SetContent(m.tabContent[m.activeTab])
				}

			}
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.run):
			m.startSpinner = true
			m.responseTime = 0
			inputUrl := m.inputs[0].Value()
			method := m.inputs[1].Value()
			headers := m.parseHeaders()
			body := m.requestBody.Value()
			cmds = append(cmds, m.spinner.Tick)
			cmds = append(cmds, doRequest(inputUrl, method, headers, body))
		case key.Matches(msg, m.keymap.addCollection):
			inputUrl := m.inputs[0].Value()
			method := m.inputs[1].Value()

			parsedUrl, err := url.Parse(inputUrl)
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}
			scheme := parsedUrl.Scheme
			host := parsedUrl.Host
			path := parsedUrl.Path
			headers := m.parseHeaders()

			if m.collectionMap == nil {
				m.collectionMap = map[string]any{
					"name":    "",
					"scheme":  scheme,
					"host":    host,
					"headers": headers,
				}
			}

			if m.collectionMap["headers"] == nil {
				m.collectionMap["headers"] = map[string]string{}
			}

			maps.Copy(m.collectionMap["headers"].(map[string]string), headers)

			if m.collectionMap[method] == nil {
				m.collectionMap[method] = map[string]any{}
			}

			if m.collectionMap[method].(map[string]any)[path] == nil {
				// TODO: add query params
				m.collectionMap[method].(map[string]any)[path] = map[string]any{}
			}

			collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}

			m.collection.SetValue(string(collectionJson))

		case key.Matches(msg, m.keymap.extractCollection):
			collectionSplit := strings.Split(m.collection.Value(), "\n")
			currentLine := collectionSplit[m.collection.Line()]
			try, err := regexp.Compile(`(\s+)"(.*)": `)
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}
			matches := try.FindStringSubmatch(currentLine)

			// TODO: find out how to get the endpoint API call structure

			if len(matches) > 2 {
				m.testExtract = fmt.Sprintf("'%s' - '%s'", matches[1], matches[2])
			}

		case key.Matches(msg, m.keymap.nextView):
			m.changeFocus()
		case key.Matches(msg, m.keymap.prevView):
			m.changeFocus()
		}

		if len(msg.String()) == 1 || msg.String() == "backspace" || msg.String() == "enter" {
			cmd := m.updateInputs(msg)
			cmds = append(cmds, cmd)
			if msg.String() == "backspace" {
				m.cursorPos -= 1
			} else if msg.String() != "enter" {
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
	m.updateCursorPos(m.cursorPos)

	m.requestHeaders.SetWidth(m.responseViewWidth)
	m.requestHeaders.SetHeight(m.responseViewHeight)

	m.collection.SetWidth(m.responseViewWidth)
	m.collection.SetHeight(m.responseViewHeight)

	m.requestBody.SetWidth(m.responseViewWidth)
	m.requestBody.SetHeight(m.responseViewHeight)

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
		if m.responseTime > 0 && i == 0 {
			fmt.Fprintf(&b, "     %d ms", m.responseTime)
		}
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	if m.statusCode > 0 {
		b.WriteString(m.statusCodeView.View())
	}

	b.WriteRune('\n')
	b.WriteString(m.testExtract)
	b.WriteRune('\n')

	b.WriteString(row)
	b.WriteRune('\n')
	switch m.activeTab {
	case TabCollection:
		b.WriteString(m.collection.View())
	case TabRequestHeaders:
		b.WriteString(m.requestHeaders.View())
	case TabRequestBody:
		b.WriteString(m.requestBody.View())
	default:
		b.WriteString(m.responseView.View())
	}
	b.WriteRune('\n')

	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.nextView,
		m.keymap.prevView,
		m.keymap.nextTab,
		m.keymap.prevTab,
		m.keymap.run,
		m.keymap.addCollection,
		m.keymap.extractCollection,
		m.keymap.quit,
	})

	b.WriteString(help)

	return b.String()
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs)+3)

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	var i int
	for i = range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	m.requestHeaders, cmds[i] = m.requestHeaders.Update(msg)
	m.requestBody, cmds[i+1] = m.requestBody.Update(msg)
	m.collection, cmds[i+2] = m.collection.Update(msg)

	return tea.Batch(cmds...)
}

func (m *model) updateCursorPos(newPos int) {
	m.cursorPos = min(newPos, 0)
	switch m.currentFocus {
	case FocusInput:
		m.cursorPos = min(newPos, len(m.inputs[m.focusInputIndex].Value()))
	case FocusResponseView:
		switch m.activeTab {
		case TabCollection:
			m.cursorPos = min(newPos, m.collection.LineInfo().CharWidth-1)
		case TabRequestHeaders:
			m.cursorPos = min(newPos, m.requestHeaders.LineInfo().CharWidth-1)
		case TabRequestBody:
			m.cursorPos = min(newPos, m.requestBody.LineInfo().CharWidth-1)
		}
	}

	m.inputs[m.focusInputIndex].SetCursor(m.cursorPos)
	m.requestHeaders.SetCursor(m.cursorPos)
	m.collection.SetCursor(m.cursorPos)
	m.requestBody.SetCursor(m.cursorPos)
}

func (m *model) changeFocus() {
	switch m.currentFocus {
	case FocusInput:
		m.currentFocus = FocusResponseView
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		switch m.activeTab {
		case TabCollection:
			m.collection.Focus()
			m.cursorPos = m.collection.LineInfo().CharWidth - 2
		case TabResponseHeaders:
			m.requestHeaders.Focus()
			m.cursorPos = m.requestHeaders.LineInfo().CharWidth - 2
		case TabRequestBody:
			m.requestBody.Focus()
			m.cursorPos = m.requestBody.LineInfo().CharWidth - 2
		}
	case FocusResponseView:
		m.currentFocus = FocusInput
		m.cursorPos = len(m.inputs[m.focusInputIndex].Value())
		m.inputs[m.focusInputIndex].Focus()
		m.collection.Blur()
		m.requestHeaders.Blur()
		m.requestBody.Blur()
	}
}

func (m *model) parseHeaders() map[string]string {
	headers := map[string]string{}
	for line := range strings.SplitSeq(m.requestHeaders.Value(), "\n") {
		lineSplit := strings.Split(line, ":")
		if len(lineSplit) > 1 {
			key := lineSplit[0]
			value := strings.TrimSpace(lineSplit[1])
			if strings.Contains(value, "{{") && strings.Contains(value, "}}") {
				start := strings.Index(value, "{{")
				end := strings.Index(value, "}}")
				if start != -1 && end != -1 && end > start {
					envVar := value[start+2 : end]
					envValue := os.Getenv(envVar)
					if envValue != "" {
						value = value[:start] + envValue + value[end+2:]
					}
				}
			}
			headers[key] = value
		}
	}

	return headers
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
		m.responseView.Style = windowStyle
	case FocusInput:
		m.inputs[m.focusInputIndex].PromptStyle = focusedStyle
		m.inputs[m.focusInputIndex].TextStyle = focusedStyle
		windowStyle = windowStyle.BorderForeground(nonHighlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(nonHighlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseView.Style = windowStyle
	}
}

func initialModel() model {
	m := model{
		help:         help.New(),
		inputs:       make([]textinput.Model, 2),
		tabs:         []string{"Collection", "Request Headers", "Request Body", "Response Body", "Response Headers"},
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
			paste: key.NewBinding(
				key.WithKeys("ctrl+v"),
				key.WithHelp("ctrl+v", "paste"),
			),
			run: key.NewBinding(
				key.WithKeys("ctrl+r"),
				key.WithHelp("ctrl+r", "run"),
			),
			addCollection: key.NewBinding(
				key.WithKeys("alt+a"),
				key.WithHelp("alt+a", "add to collection"),
			),
			extractCollection: key.NewBinding(
				key.WithKeys("alt+e"),
				key.WithHelp("alt+e", "extract from collection"),
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

	m.responseView = viewport.New(78, 20)
	m.responseView.Style = windowStyle

	m.collection = textarea.New()
	m.collection.Cursor.Style = cursorStyle
	m.collection.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.collection.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestHeaders = textarea.New()
	m.requestHeaders.Cursor.Style = cursorStyle
	m.requestHeaders.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.requestHeaders.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestBody = textarea.New()
	m.requestBody.Cursor.Style = cursorStyle
	m.requestBody.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.requestBody.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.statusCodeView = viewport.New(16, 1)
	m.statusCodeView.Style = statusCodeViewStyle

	return m
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right

	return border
}
