package postui

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/atotto/clipboard"
	genmock "github.com/bramca/gen-mockserver"
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
	placeHolderUrl    = "https://example.com/api/v1/something"
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

	jsonRegex       *regexp.Regexp
	queryParamRegex *regexp.Regexp
)

type keymap = struct {
	nextView, prevView, nextTab, prevTab, left, right, up, down, j, k, l, h, paste, copy, save, run, addCollection, extractCollection, quit key.Binding
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
	statusCode         int
	responseTime       int64
	err                error
	startSpinner       bool
	responseBody       string
	responseHeaders    string
	collectionFilePath string
	tabs               []string
	tabContent         []string
	collectionMap      map[string]any
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.tabContent[TabCollection] = m.collection.Value()
	m.tabContent[TabRequestBody] = m.requestBody.Value()
	m.tabContent[TabRequestHeaders] = m.requestHeaders.Value()

	switch msg := msg.(type) {
	case responseMsg:
		m.startSpinner = false
		m.err = nil
		m.currentFocus = FocusResponseView
		m.statusCode = msg.statusCode
		m.activeTab = TabResponseBody
		m.collection.Blur()
		m.requestBody.Blur()
		m.requestHeaders.Blur()
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
		m.currentFocus = FocusResponseView
		m.activeTab = TabResponseBody
		m.err = msg
		m.tabContent[m.activeTab] = m.err.Error()
		m.responseView.SetContent(m.tabContent[m.activeTab])
		m.changeFocus()

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
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollUp(1)
			}

		case key.Matches(msg, m.keymap.down):
			if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
				m.responseView.ScrollDown(1)
			}

		case key.Matches(msg, m.keymap.copy):
			err := clipboard.WriteAll(m.tabContent[m.activeTab])
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
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
				cursorPos := currentInput.Position()
				if cursorPos >= len(currentInput.Value())-1 || cursorPos <= 0 {
					m.inputs[m.focusInputIndex].SetValue(currentInput.Value() + cb)
				} else {
					m.inputs[m.focusInputIndex].SetValue(currentInput.Value()[0:cursorPos] + cb + currentInput.Value()[cursorPos:len(currentInput.Value())-1])
				}

				m.inputs[m.focusInputIndex].SetCursor(cursorPos + len(cb))
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

		case key.Matches(msg, m.keymap.save):
			currentCollection := map[string]any{}
			err := json.Unmarshal([]byte(m.collection.Value()), &currentCollection)
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}

			maps.Copy(m.collectionMap, currentCollection)

			if m.collectionFilePath != "" {
				err := os.WriteFile(m.collectionFilePath, []byte(m.collection.Value()), 0o644)
				if err != nil {
					return m, func() tea.Msg {
						return errMsg{err: err}
					}
				}
			} else if filename, ok := m.collectionMap["filename"].(string); ok && filename != "" {
				err := os.WriteFile(filename, []byte(m.collection.Value()), 0o644)
				if err != nil {
					return m, func() tea.Msg {
						return errMsg{err: err}
					}
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
			queryParameters := parsedUrl.Query()
			body := map[string]any{}
			if m.requestBody.Value() != "" {
				err := json.Unmarshal([]byte(m.requestBody.Value()), &body)
				if err != nil {
					return m, func() tea.Msg {
						return errMsg{err: err}
					}
				}
			}

			m.addToCollectionMap(scheme, host, method, path, body, queryParameters, headers)

			collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
			if err != nil {
				return m, func() tea.Msg {
					return errMsg{err: err}
				}
			}

			m.collection.SetValue(string(collectionJson))

			m.activeTab = TabCollection
			if m.currentFocus != FocusResponseView {
				m.changeFocus()
			}

		case key.Matches(msg, m.keymap.extractCollection):
			collectionSplit := strings.Split(m.collection.Value(), "\n")
			currentLine := collectionSplit[m.collection.Line()]
			matches := jsonRegex.FindStringSubmatch(currentLine)

			if len(matches) > 2 {
				method := ""
				endpoint := matches[2]
				filter := ""
				headers, ok := m.collectionMap["headers"].(map[string]any)
				indentation := matches[1]
				i := m.collection.Line() - 1
				for i >= 0 {
					currentLine = collectionSplit[i]
					matches = jsonRegex.FindStringSubmatch(currentLine)
					if len(matches) > 2 {
						parentIndentation := matches[1]
						parentKey := matches[2]
						if len(parentIndentation) < len(indentation) {
							if slices.Contains([]string{http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodPatch, http.MethodPost, http.MethodPut}, parentKey) {
								method = parentKey
								break
							} else {
								// We were actually in a filter and not an endpoint
								filter = endpoint
								endpoint = parentKey
								indentation = parentIndentation
							}
						}
					}
					i--
				}

				if ok {
					for header, value := range headers {
						headerText := fmt.Sprintf("%s: %s", header, value)
						if !strings.Contains(m.requestHeaders.Value(), headerText) {
							newline := ""
							if m.requestHeaders.Value() != "" {
								newline = "\n"
							}
							m.requestHeaders.SetValue(m.requestHeaders.Value() + newline + headerText)
						}
					}
				}

				if method != "" {
					isWriteMethod := slices.Contains([]string{http.MethodPatch, http.MethodPost, http.MethodPut}, method)
					urlText := fmt.Sprintf("%s://%s%s", m.collectionMap["scheme"], m.collectionMap["host"], endpoint)
					matches := queryParamRegex.FindStringSubmatch(m.inputs[0].Value())
					if len(matches) > 2 && matches[1] == urlText {
						urlText = m.inputs[0].Value()
					}
					if filter != "" && !isWriteMethod {
						if strings.Contains(urlText, "=") {
							urlText += "&" + filter + "="
						} else {
							urlText += "?" + filter + "="
						}
					}

					if isWriteMethod {
						requestBody := m.collectionMap[method].(map[string]any)[endpoint].(map[string]any)["body"]
						if requestBody != nil {
							requestBodyJson, err := json.MarshalIndent(requestBody, "", "  ")
							if err != nil {
								return m, func() tea.Msg {
									return errMsg{err: err}
								}
							}
							m.requestBody.SetValue(string(requestBodyJson))
						}
					}

					m.inputs[0].SetValue(urlText)
					m.inputs[1].SetValue(method)
				}
			}

		case key.Matches(msg, m.keymap.nextView):
			m.changeFocus()
		case key.Matches(msg, m.keymap.prevView):
			m.changeFocus()
		}

		if len(msg.String()) == 1 || slices.Contains([]string{"backspace", "enter", "up", "down", "left", "right"}, msg.String()) {
			cmd := m.updateInputs(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder
	var renderedTabs []string

	m.updateFocusView()

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
		m.keymap.copy,
		m.keymap.save,
		m.keymap.quit,
	})

	b.WriteString(help)

	return b.String()
}

func (m *model) addToCollectionMap(scheme string, host string, method string, path string, body any, queryParameters url.Values, headers map[string]string) {
	if m.collectionMap == nil {
		m.collectionMap = map[string]any{
			"name":     "",
			"filename": "",
			"scheme":   scheme,
			"host":     host,
			"headers":  headers,
		}
	}

	if _, ok := m.collectionMap["headers"].(map[string]string); !ok {
		m.collectionMap["headers"] = map[string]string{}
	}

	maps.Copy(m.collectionMap["headers"].(map[string]string), headers)

	if method == "" {
		return
	}

	if m.collectionMap[method] == nil {
		m.collectionMap[method] = map[string]any{}
	}

	if m.collectionMap[method].(map[string]any)[path] == nil {
		m.collectionMap[method].(map[string]any)[path] = map[string]any{}
	}

	if slices.Contains([]string{http.MethodPatch, http.MethodPost, http.MethodPut}, method) {
		m.collectionMap[method].(map[string]any)[path].(map[string]any)["body"] = body
	}

	for param, value := range queryParameters {
		m.collectionMap[method].(map[string]any)[path].(map[string]any)[param] = value
	}
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
		case TabResponseHeaders:
			m.requestHeaders.Focus()
		case TabRequestBody:
			m.requestBody.Focus()
		}
	case FocusResponseView:
		m.currentFocus = FocusInput
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

func InitialModel(collectionFilePath string, specFile string, specVersion int) model {
	m := model{
		help:               help.New(),
		inputs:             make([]textinput.Model, 2),
		tabs:               []string{"Collection", "Request Headers", "Request Body", "Response Body", "Response Headers"},
		currentFocus:       FocusInput,
		spinner:            spinner.New(),
		collectionFilePath: collectionFilePath,
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
			copy: key.NewBinding(
				key.WithKeys("alt+x"),
				key.WithHelp("alt+x", "copy"),
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
			quit: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "quit"),
			),
		},
	}

	var err error
	jsonRegex, err = regexp.Compile(`(\s+)"(.*)": `)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong with compiling the header regex: %v", err)
		os.Exit(2)
	}

	queryParamRegex, err = regexp.Compile(`(.*)\?(.*)=`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong with compiling the query parameter regex: %v", err)
		os.Exit(2)
	}

	if m.collectionFilePath != "" {
		collectionFile, err := os.ReadFile(m.collectionFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Something went wrong with reading the file '%s': %v", m.collectionFilePath, err)
			os.Exit(2)
		}

		err = json.Unmarshal([]byte(collectionFile), &m.collectionMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Something went wrong with parsing the file '%s': %v", m.collectionFilePath, err)
			os.Exit(2)
		}
	}

	if specFile != "" {
		specDataStructure := map[string]map[string][]genmock.RequestStructure{}
		if specVersion == 2 {
			specDataStructure = genmock.SpecV2toRequestStructureMap(specFile, 1, false)
		}
		if specVersion == 3 {
			specDataStructure = genmock.SpecV3toRequestStructureMap(specFile, 1, false)
		}

		for method, calls := range specDataStructure {
			for path, requestStructures := range calls {
				queryParams := url.Values{}
				var body any
				for _, requestStructure := range requestStructures {
					body = requestStructure.RequestBody
					matches := queryParamRegex.FindStringSubmatch(requestStructure.Path)
					if len(matches) > 2 {
						queryParams[matches[2]] = []string{}
					}
				}
				m.addToCollectionMap("", "", strings.ToUpper(method), path, body, queryParams, map[string]string{})
			}
		}
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
			t.Placeholder = placeHolderUrl
			t.Width = t.CharLimit
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		// Headers
		case 1:
			t.CharLimit = 10
			t.Placeholder = placeHolderMethod
			t.Width = t.CharLimit
		}

		m.inputs[i] = t
	}

	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinner.Moon

	m.responseView = viewport.New(78, 20)
	m.responseView.Style = windowStyle

	m.collection = textarea.New()
	if m.collectionMap != nil {
		collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Something went wrong with parsing the file '%s': %v", m.collectionFilePath, err)
			os.Exit(2)
		}
		m.collection.SetValue(string(collectionJson))
	}
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
