package postui

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) handleResponseMsg(msg responseMsg) {
	m.startSpinner = false
	m.err = nil
	m.currentFocus = FocusBottom
	m.statusCode = msg.statusCode
	m.activeTab = TabResponseBody
	m.collectionEdit.Blur()
	m.requestBody.Blur()
	m.requestHeaders.Blur()
	m.responseBody = msg.responseBody
	m.responseHeaders = msg.responseHeaders
	m.responseTime = msg.responseTime

	m.tabContent[TabResponseBody] = m.responseBody
	m.tabContent[TabResponseHeaders] = m.responseHeaders
	m.responseView.SetContent(m.tabContent[m.activeTab])

	if m.statusCode > 0 {
		statusMsgExtra := ""
		if m.statusCode < 300 {
			statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
		}

		if m.statusCode > 299 && m.statusCode < 400 {
			statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#FFC66D"})
		}

		if m.statusCode > 399 {
			statusMsgExtra = ""
			statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#DA4939"})
		}
		statusMsg := fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode))
		responseTimeMsg := fmt.Sprintf("%d ms", m.responseTime)
		if m.responseTime > 1000 {
			responseTimeMsg = fmt.Sprintf("%d s", m.responseTime/1000)
		}
		padding := (m.statusCodeView.Width - len(statusMsg)) / 2
		paddingRespTime := (m.responseTimeView.Width - len(responseTimeMsg)) / 2
		statusCodeContent := fmt.Sprintf(" %s  %s%s", statusMsgExtra, strings.Repeat(" ", max(0, padding-5)), statusMsg)
		if len(statusCodeContent) >= inputWidthPadding {
			newStatusCodeContent := make([]byte, inputWidthPadding)
			for i := range inputWidthPadding - 4 {
				newStatusCodeContent[i] = statusCodeContent[i]
			}
			statusCodeContent = fmt.Sprintf("%s..", string(newStatusCodeContent))
		}
		m.responseTimeView.SetContent(fmt.Sprintf("  %s%s", strings.Repeat(" ", max(0, paddingRespTime-4)), responseTimeMsg))
		m.statusCodeView.SetContent(statusCodeContent)
		m.statusCodeView.Style = statusCodeViewStyle
	}

	for i := range m.inputs {
		// Remove focus from inputs
		m.inputs[i].Blur()
	}
}

func (m *model) handleErrMsg(msg errMsg) {
	m.startSpinner = false
	m.statusCode = 0
	m.responseTime = 0
	m.responseBody = ""
	m.activeTab = TabResponseBody
	m.err = msg
	m.tabContent[m.activeTab] = m.err.Error()
	m.responseView.SetContent(m.tabContent[m.activeTab])
	if m.currentFocus != FocusBottom {
		m.changeFocus()
	}
}

func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	m.responseViewWidth = msg.Width
	m.responseViewHeight = msg.Height - paddingHeight

	windowStyle = windowStyle.Width(m.responseViewWidth).Height(m.responseViewHeight)

	for i := range m.inputs {
		m.inputs[i].Width = m.responseViewWidth - inputWidthPadding
	}

	m.responseView.Width = m.responseViewWidth
	m.responseView.Height = m.responseViewHeight

	m.collectionView.Width = m.responseViewWidth
	m.collectionView.Height = m.responseViewHeight

	m.requestHeaders.SetWidth(m.responseViewWidth)
	m.requestHeaders.SetHeight(m.responseViewHeight)

	m.collectionEdit.SetWidth(m.responseViewWidth)
	m.collectionEdit.SetHeight(m.responseViewHeight)

	m.requestBody.SetWidth(m.responseViewWidth)
	m.requestBody.SetHeight(m.responseViewHeight)

	m.responseView.Style = windowStyle

	m.collectionList.SetSize(m.responseViewWidth, m.responseViewHeight)

	m.updateFocusView()
}

func (m *model) handleKeyMsg(msg tea.KeyMsg, cmds []tea.Cmd) ([]tea.Cmd, error) {
	if m.notify {
		m.notify = false
	}

	switch {
	case key.Matches(msg, m.keymap.help):
		m.help.ShowAll = !m.help.ShowAll
	case key.Matches(msg, m.keymap.h):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollLeft(1)
		}

	case key.Matches(msg, m.keymap.j):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollDown(1)
		}

	case key.Matches(msg, m.keymap.k):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollUp(1)
		}

	case key.Matches(msg, m.keymap.l):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollRight(1)
		}

	case key.Matches(msg, m.keymap.up):
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollUp(1)
		} else {
			switch m.activeTab {
			case TabCollection:
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: -2})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: -2})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: -2})
				cmds = append(cmds, cmd)
			}
		}

	case key.Matches(msg, m.keymap.down):
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollDown(1)
		} else {
			switch m.activeTab {
			case TabCollection:
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: -3})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: -3})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: -3})
				cmds = append(cmds, cmd)
			}
		}

	case key.Matches(msg, m.keymap.scrollUpMulti):
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollUp(multiScrollSize)
		} else {
			switch m.activeTab {
			case TabCollection:
				for range multiScrollSize {
					m.collectionEdit.CursorUp()
				}
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: -2})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				for range multiScrollSize {
					m.requestBody.CursorUp()
				}
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: -2})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				for range multiScrollSize {
					m.requestHeaders.CursorUp()
				}
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: -2})
				cmds = append(cmds, cmd)
			}
		}

	case key.Matches(msg, m.keymap.scrollDownMulti):
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollDown(20)
		} else {
			switch m.activeTab {
			case TabCollection:
				for range 19 {
					m.collectionEdit.CursorDown()
				}
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: -3})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				for range 19 {
					m.requestBody.CursorDown()
				}
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: -3})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				for range 19 {
					m.requestHeaders.CursorDown()
				}
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: -3})
				cmds = append(cmds, cmd)
			}
		}

	case key.Matches(msg, m.keymap.toggleCollectionEdit):
		if m.activeTab == TabCollection {
			switch m.collectionType {
			case CollectionEdit:
				m.collectionType = CollectionList
				m.collectionEdit.Blur()
			case CollectionList:
				m.collectionType = CollectionEdit
				m.collectionEdit.Focus()
			}
		}

	case key.Matches(msg, m.keymap.collectionListSelect):
		if m.currentFocus == FocusBottom && m.activeTab == TabCollection && m.collectionType == CollectionList && m.collectionList.SelectedItem() != nil {
			collectionKey := m.collectionList.SelectedItem().FilterValue()
			// if the previous selectedItem was a HTTP method, we now selected a path
			method := ""
			endpoint := m.requestEndpoint
			filter := ""
			if slices.Contains([]string{http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodPatch, http.MethodPost, http.MethodPut}, m.selectedItem) {
				method = m.selectedItem
				endpoint = collectionKey
			} else if m.selectedItem == "servers" {
				parseServer, err := url.Parse(collectionKey)
				if err != nil {
					return nil, err
				}
				m.requestHost = parseServer.Host
				m.requestBasePath = parseServer.Path
				m.requestScheme = parseServer.Scheme

				urlText := fmt.Sprintf("%s://%s%s%s", m.requestScheme, m.requestHost, m.requestBasePath, m.requestEndpoint)

				m.inputs[0].SetValue(urlText)
			} else if m.requestEndpoint != "" && m.selectedItem != "" {
				method = m.inputs[1].Value()
				endpoint = m.selectedItem
				filter = collectionKey
			}

			if method != "" && endpoint != "" {
				err := m.setRequestInputs(method, endpoint, filter)
				if err != nil {
					return nil, err
				}
			}
			collectionMap := m.collectionMap
			if method != "" {
				collectionMap = m.collectionMap[method].(map[string]any)
			}
			m.setCollectionList(collectionMap, collectionKey, collectionKey)
		}

	case key.Matches(msg, m.keymap.collectionListReturn):
		if m.currentFocus == FocusBottom && m.activeTab == TabCollection && m.collectionType == CollectionList && len(m.previousItems) > 0 {
			m.selectedItem = m.previousItems[len(m.previousItems)-1]
			if len(m.previousItems)-1 > 0 {
				m.previousItems = m.previousItems[:len(m.previousItems)-1]
			}
			m.setCollectionList(m.collectionMap, m.selectedItem, m.selectedItem)
		}

	case key.Matches(msg, m.keymap.focusCollection):
		m.currentFocus = FocusBottom
		m.activeTab = TabCollection
		m.changeActiveTab()

	case key.Matches(msg, m.keymap.top):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.GotoTop()
		}

	case key.Matches(msg, m.keymap.bottom):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.GotoBottom()
		}

	case key.Matches(msg, m.keymap.copy):
		err := clipboard.WriteAll(m.tabContent[m.activeTab])
		if err != nil {
			return nil, err
		}

	case key.Matches(msg, m.keymap.copyCurl):
		err := clipboard.WriteAll(m.copyCurl())
		if err != nil {
			return nil, err
		}

	case key.Matches(msg, m.keymap.paste):
		cb, err := clipboard.ReadAll()
		if err != nil {
			return nil, err
		}
		switch m.currentFocus {
		case FocusTop:
			currentInput := m.inputs[m.focusInputIndex]
			cursorPos := currentInput.Position()
			if cursorPos >= len(currentInput.Value())-1 || cursorPos <= 0 {
				m.inputs[m.focusInputIndex].SetValue(currentInput.Value() + cb)
			} else {
				m.inputs[m.focusInputIndex].SetValue(currentInput.Value()[0:cursorPos] + cb + currentInput.Value()[cursorPos:len(currentInput.Value())])
			}

			m.inputs[m.focusInputIndex].SetCursor(cursorPos + len(cb))
		case FocusBottom:
			switch m.activeTab {
			case TabCollection:
				m.collectionEdit.InsertString(cb)
			case TabRequestHeaders:
				m.requestHeaders.InsertString(cb)
			case TabRequestBody:
				m.requestBody.InsertString(cb)
			}
		}

	case key.Matches(msg, m.keymap.save):
		currentCollection := map[string]any{}
		err := json.Unmarshal([]byte(m.collectionEdit.Value()), &currentCollection)
		if err != nil {
			return nil, err
		}

		// to prevent the servers being of type []any
		if collectionServers, ok := currentCollection["servers"].([]any); ok {
			servers := []string{}
			for _, server := range collectionServers {
				servers = append(servers, server.(string))
			}

			currentCollection["servers"] = servers
		}

		maps.Copy(m.collectionMap, currentCollection)

		if m.collectionFilePath != "" {
			err := os.WriteFile(m.collectionFilePath, []byte(m.collectionEdit.Value()), 0o644)
			if err != nil {
				return nil, err
			}
		} else if filename, ok := m.collectionMap["filename"].(string); ok && filename != "" {
			err := os.WriteFile(filename, []byte(m.collectionEdit.Value()), 0o644)
			if err != nil {
				return nil, err
			}
		}
		m.setCollectionList(m.collectionMap, "", "")
		m.selectedItem = ""

		statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
		statusCodeContent := "      Saved"
		m.statusCodeView.SetContent(statusCodeContent)
		m.statusCodeView.Style = statusCodeViewStyle
		m.notify = true

	case key.Matches(msg, m.keymap.nextTab), key.Matches(msg, m.keymap.prevTab):
		s := msg.String()
		switch m.currentFocus {
		case FocusTop:
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
					m.collectionEdit.Blur()
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
		case FocusBottom:
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
				m.collectionEdit.Focus()
				m.requestBody.Blur()
				m.requestHeaders.Blur()
			case TabRequestHeaders:
				m.requestHeaders.Focus()
				m.requestBody.Blur()
				m.collectionEdit.Blur()
			case TabRequestBody:
				m.requestBody.Focus()
				m.requestHeaders.Blur()
				m.collectionEdit.Blur()
			default:
				m.requestHeaders.Blur()
				m.collectionEdit.Blur()
				m.requestBody.Blur()
				m.responseView.SetContent(m.tabContent[m.activeTab])
			}

		}

	case key.Matches(msg, m.keymap.quit):
		return []tea.Cmd{tea.Quit}, nil

	case key.Matches(msg, m.keymap.run):
		m.startSpinner = true
		m.responseTime = 0
		inputUrl := m.inputs[0].Value()
		method := m.inputs[1].Value()
		headers := m.parseHeaders()
		body := m.requestBody.Value()
		query := m.inputs[2].Value()

		parsedUrl, err := url.Parse(inputUrl)
		if err != nil {
			return nil, err
		}
		m.requestHost = parsedUrl.Host
		m.requestScheme = parsedUrl.Scheme
		rawQuery := ""
		if parsedUrl.RawQuery != "" {
			rawQuery = "?" + parsedUrl.RawQuery
		}
		m.requestEndpoint = strings.Replace(strings.TrimSpace(parsedUrl.Path+rawQuery), m.requestBasePath, "", 1)

		cmds = append(cmds, m.spinner.Tick)
		cmds = append(cmds, doRequest(inputUrl, method, headers, body, query))

	case key.Matches(msg, m.keymap.addCollection):
		inputUrl := m.inputs[0].Value()
		method := m.inputs[1].Value()

		parsedUrl, err := url.Parse(inputUrl)
		if err != nil {
			return nil, err
		}
		scheme := parsedUrl.Scheme
		host := parsedUrl.Host
		path := strings.Replace(parsedUrl.Path, m.requestBasePath, "", 1)
		headers := map[string]string{}
		for line := range strings.SplitSeq(m.requestHeaders.Value(), "\n") {
			lineSplit := strings.Split(line, ":")
			if len(lineSplit) > 1 {
				key := lineSplit[0]
				value := strings.TrimSpace(lineSplit[1])
				headers[key] = value
			}
		}
		queryParameters := parsedUrl.Query()
		body := map[string]any{}
		if m.requestBody.Value() != "" {
			err := json.Unmarshal([]byte(m.requestBody.Value()), &body)
			if err != nil {
				return nil, err
			}
		}

		m.requestHost = host
		m.requestScheme = scheme
		rawQuery := ""
		if parsedUrl.RawQuery != "" {
			rawQuery = "?" + parsedUrl.RawQuery
		}
		m.requestEndpoint = strings.Replace(parsedUrl.Path+rawQuery, m.requestBasePath, "", 1)

		m.addToCollectionMap(scheme, host, method, path, body, queryParameters, headers)

		collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
		if err != nil {
			return nil, err
		}

		m.collectionEdit.SetValue(string(collectionJson))
		m.setCollectionList(m.collectionMap, "", "")
		m.selectedItem = ""

		m.activeTab = TabCollection
		if m.currentFocus != FocusBottom {
			m.changeFocus()
		} else {
			m.changeActiveTab()
		}

		statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
		statusCodeContent := "      Added"
		m.statusCodeView.SetContent(statusCodeContent)
		m.notify = true

	case key.Matches(msg, m.keymap.extractCollection):
		collectionSplit := strings.Split(m.collectionEdit.Value(), "\n")
		currentLine := collectionSplit[m.collectionEdit.Line()]
		matches := m.jsonRegex.FindStringSubmatch(currentLine)

		if len(matches) > 2 {
			method := ""
			endpoint := matches[2]
			filter := ""
			indentation := matches[1]
			i := m.collectionEdit.Line() - 1
			for i >= 0 {
				currentLine = collectionSplit[i]
				matches = m.jsonRegex.FindStringSubmatch(currentLine)
				if len(matches) > 2 {
					parentIndentation := matches[1]
					parentKey := matches[2]
					if len(parentIndentation) < len(indentation) {
						if slices.Contains([]string{http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodPatch, http.MethodPost, http.MethodPut}, parentKey) {
							method = parentKey
							break
						} else if parentKey == "servers" {
							parseServer, err := url.Parse(endpoint)
							if err != nil {
								return nil, err
							}
							m.requestHost = parseServer.Host
							m.requestBasePath = parseServer.Path
							m.requestScheme = parseServer.Scheme

							urlText := fmt.Sprintf("%s://%s%s%s", m.requestScheme, m.requestHost, m.requestBasePath, m.requestEndpoint)
							m.inputs[0].SetValue(urlText)

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

			err := m.setRequestInputs(method, endpoint, filter)
			if err != nil {
				return nil, err
			}
		}

	case key.Matches(msg, m.keymap.extractHeaders):
		headers, headersOk := m.collectionMap["headers"].(map[string]any)
		if headersOk {
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

	case key.Matches(msg, m.keymap.nextView):
		m.changeFocus()
	case key.Matches(msg, m.keymap.prevView):
		m.changeFocus()
	}

	if len(msg.String()) == 1 || slices.Contains([]string{"backspace", "enter", "up", "down", "left", "right", "ctrl+a", "ctrl+e"}, msg.String()) {
		cmd := m.updateInputs(msg)
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}
