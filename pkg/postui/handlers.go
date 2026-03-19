package postui

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
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
	m.responseSize = msg.responseSize

	m.tabContent[TabResponseBody] = m.responseBody
	m.tabContent[TabResponseHeaders] = m.responseHeaders
	m.responseView.SetContent(m.tabContent[m.activeTab])

	if m.statusCode > 0 {
		m.setResponseStatusViews()
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
		if m.statusCode > 0 {
			m.setResponseStatusViews()
		}
	}

	switch {
	case key.Matches(msg, m.keymap.help):
		m.help.ShowAll = !m.help.ShowAll
	case key.Matches(msg, m.keymap.goBack):
		// Scrolling left when in response view tab
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollLeft(1)
		}

		// Go back in list when in collection list view tab
		if m.currentFocus == FocusBottom && m.activeTab == TabCollection && m.collectionType == CollectionList && !m.collectionList.FilterInput.Focused() {
			if len(m.previousItems) > 0 {
				m.selectedItem = m.previousItems[len(m.previousItems)-1]
				m.previousItems = m.previousItems[:len(m.previousItems)-1]
				collectionMap := m.collectionMap
				for _, item := range m.previousItems {
					if _, ok := collectionMap[item].(map[string]any); ok {
						collectionMap = collectionMap[item].(map[string]any)
					}
				}
				m.setCollectionList(collectionMap, m.selectedItem, m.selectedItem)
				m.selectedFilter = ""
				m.collectionList.ResetFilter()
				m.collectionList.KeyMap.NextPage.SetEnabled(false)
			} else if m.collectionDir != "" && m.collectionSelected {
				_, err := os.Stat(m.collectionDir)
				if err == nil {
					m.readCollectionDir()
					m.setCollectionList(m.collectionMap, "", "")
					m.collectionFilePath = ""
					m.collectionSelected = false
				}
			}
		}

	case key.Matches(msg, m.keymap.goDown):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollDown(1)
		}

	case key.Matches(msg, m.keymap.goUp):
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollUp(1)
		}

	case key.Matches(msg, m.keymap.goForward):
		// Scrolling right when in response view tab
		if m.currentFocus == FocusBottom && (m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders) {
			m.responseView.ScrollRight(1)
		}

		// Select list item when in collection list view tab
		if m.currentFocus == FocusBottom && m.activeTab == TabCollection && m.collectionType == CollectionList && m.collectionList.SelectedItem() != nil && !m.collectionList.FilterInput.Focused() {
			collectionKey := m.collectionList.SelectedItem().FilterValue()
			// if the previous selectedItem was a HTTP method, we now selected a path
			method := ""
			endpoint := m.requestEndpoint
			filter := ""
			filterValue := ""
			collectionMap := m.collectionMap
			if slices.Contains([]string{http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodPatch, http.MethodPost, http.MethodPut}, m.selectedItem) {
				method = m.selectedItem
				endpoint = collectionKey
				collectionMap = m.collectionMap[method].(map[string]any)
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
			} else if m.selectedItem != "" && m.selectedItem != "headers" && m.requestEndpoint != "" && m.selectedFilter == "" {
				method = m.inputs[1].Value()
				endpoint = m.selectedItem
				filter = collectionKey

				collectionMap = m.collectionMap[method].(map[string]any)[endpoint].(map[string]any)
			} else if m.selectedItem != "" && m.selectedItem != "headers" && m.requestEndpoint != "" && m.selectedFilter != "" {
				method = m.inputs[1].Value()
				endpoint = m.requestEndpoint
				filterValue = collectionKey
			} else if m.selectedItem == "headers" {
				headers, headersOk := m.collectionMap["headers"].(map[string]any)
				if headersOk {
					selectedHeaderValue, selectedHeaderOk := headers[collectionKey]
					if selectedHeaderOk {
						headerText := fmt.Sprintf("%s: %s", collectionKey, selectedHeaderValue)
						newline := ""
						if m.requestHeaders.Value() != "" {
							newline = "\n"
						}
						if !strings.Contains(m.requestHeaders.Value(), headerText) {
							m.requestHeaders.SetValue(m.requestHeaders.Value() + newline + headerText)
						}
					}
				}
			} else if m.collectionDir != "" && !m.collectionSelected {
				m.collectionFilePath = path.Join(m.collectionDir, m.collectionMap[collectionKey].(string))
				m.readCollectionFile()

				collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
				if err != nil {
					return nil, err
				}

				m.collectionEdit.SetValue(string(collectionJson))

				m.setCollectionList(m.collectionMap, "", "")
				m.inputs[0].SetValue("")
				m.inputs[1].SetValue("")
				m.requestHost = ""
				m.requestEndpoint = ""
				m.collectionSelected = true
			}

			if method != "" && endpoint != "" {
				err := m.setRequestInputs(method, endpoint, filter, filterValue)
				if err != nil {
					return nil, err
				}
			}

			m.setCollectionList(collectionMap, collectionKey, collectionKey)
			if filter != "" && collectionKey == m.selectedItem {
				m.selectedFilter = filter
			}
			m.collectionList.ResetFilter()
			m.collectionList.KeyMap.NextPage.SetEnabled(false)
		}

	case key.Matches(msg, m.keymap.up):
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollUp(1)
		} else {
			switch m.activeTab {
			case TabCollection:
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: tea.KeyUp})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: tea.KeyUp})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				// for correctly updating the viewport view
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: tea.KeyUp})
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
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: tea.KeyDown})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: tea.KeyDown})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: tea.KeyDown})
				cmds = append(cmds, cmd)
			}
		}

	case key.Matches(msg, m.keymap.left):
		if m.currentFocus == FocusTop {
			m.inputs[m.focusInputIndex].SetCursor(m.inputs[m.focusInputIndex].Position() - 1)
		}
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollLeft(1)
		} else {
			switch m.activeTab {
			case TabCollection:
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: tea.KeyLeft})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: tea.KeyLeft})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: tea.KeyLeft})
				cmds = append(cmds, cmd)
			}
		}

	case key.Matches(msg, m.keymap.right):
		if m.currentFocus == FocusTop {
			m.inputs[m.focusInputIndex].SetCursor(m.inputs[m.focusInputIndex].Position() + 1)
		}
		if m.activeTab == TabResponseBody || m.activeTab == TabResponseHeaders {
			m.responseView.ScrollRight(1)
		} else {
			switch m.activeTab {
			case TabCollection:
				var cmd tea.Cmd
				m.collectionEdit, cmd = m.collectionEdit.Update(tea.KeyMsg{Type: tea.KeyRight})
				cmds = append(cmds, cmd)

			case TabRequestBody:
				var cmd tea.Cmd
				m.requestBody, cmd = m.requestBody.Update(tea.KeyMsg{Type: tea.KeyRight})
				cmds = append(cmds, cmd)

			case TabRequestHeaders:
				var cmd tea.Cmd
				m.requestHeaders, cmd = m.requestHeaders.Update(tea.KeyMsg{Type: tea.KeyRight})
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

		statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
		statusCodeContent := "     Copied"
		m.statusCodeView.SetContent(statusCodeContent)
		m.statusCodeView.Style = statusCodeViewStyle
		m.notify = true

	case key.Matches(msg, m.keymap.copyCurl):
		err := clipboard.WriteAll(m.copyCurl())
		if err != nil {
			return nil, err
		}
		statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
		statusCodeContent := "   Curl Copied"
		m.statusCodeView.SetContent(statusCodeContent)
		m.statusCodeView.Style = statusCodeViewStyle
		m.notify = true

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
			filePath, err := ExpandPath(m.collectionFilePath)
			if err != nil {
				return nil, err
			}
			err = AtomicWrite(filePath, []byte(m.collectionEdit.Value()), 0o644)
			if err != nil {
				return nil, err
			}
		} else if filename, ok := m.collectionMap["filename"].(string); ok && filename != "" {
			filePath := filename
			if m.collectionDir != "" {
				filePath = path.Join(m.collectionDir, filename)
			}

			filePath, err = ExpandPath(filePath)
			if err != nil {
				return nil, err
			}

			err = AtomicWrite(filePath, []byte(m.collectionEdit.Value()), 0o644)
			if err != nil {
				return nil, err
			}
		}
		collectionMap := m.collectionMap
		for _, item := range m.previousItems {
			if _, ok := collectionMap[item].(map[string]any); ok {
				collectionMap = collectionMap[item].(map[string]any)
			}
		}
		m.setCollectionList(collectionMap, m.selectedItem, "")

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
				if m.collectionType == CollectionEdit {
					m.collectionEdit.Focus()
				}
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
		m.statusCode = 0
		m.responseSize = 0
		inputUrl := m.inputs[0].Value()
		method := m.inputs[1].Value()
		headers := m.parseHeaders(false)
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
		collectionMap := m.collectionMap
		for _, item := range m.previousItems {
			if _, ok := collectionMap[item].(map[string]any); ok {
				collectionMap = collectionMap[item].(map[string]any)
			}
		}
		m.setCollectionList(collectionMap, m.selectedItem, "")

		m.activeTab = TabCollection
		if m.currentFocus != FocusBottom {
			m.changeFocus()
		} else {
			m.changeActiveTab()
		}

		statusCodeViewStyle = statusCodeViewStyle.Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"})
		statusCodeContent := "      Added"
		m.statusCodeView.SetContent(statusCodeContent)
		m.statusCodeView.Style = statusCodeViewStyle
		m.notify = true

	case key.Matches(msg, m.keymap.nextView):
		m.changeFocus()
	case key.Matches(msg, m.keymap.prevView):
		m.changeFocus()
	case key.Matches(msg, m.keymap.reloadConfig):
		var err error
		defaultConfigPath, err := ExpandPath(m.config.DefaultCollectionDir)
		if err != nil {
			return nil, err
		}

		m.config, err = NewConfig()
		if err != nil {
			return nil, err
		}

		m.applyConfig(m.collectionDir == defaultConfigPath)
	}

	if msg.String() == "backspace" && m.currentFocus == FocusBottom && m.activeTab == TabCollection && m.collectionType == CollectionList {
		return cmds, nil
	}

	if len(msg.String()) == 1 || slices.Contains([]string{"backspace", "enter", "up", "down", "left", "right", "ctrl+a", "ctrl+e"}, msg.String()) {
		cmd := m.updateInputs(msg)
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}
