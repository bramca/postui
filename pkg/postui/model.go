package postui

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	genmock "github.com/bramca/gen-mockserver"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pb33f/libopenapi"
)

type (
	Focus          int
	Tab            int
	CollectionType int
)

const (
	FocusTop Focus = iota
	FocusBottom
)

const (
	CollectionList CollectionType = iota
	CollectionEdit
)

const (
	TabCollection Tab = iota
	TabRequestHeaders
	TabRequestBody
	TabResponseBody
	TabResponseHeaders
)

const (
	placeHolderUrl    = "https://example.com/api/v1/object"
	placeHolderMethod = "GET"
	placeHolderJq     = ".data[].object.property"
	paddingHeight     = 8
	inputWidthPadding = 21
	multiScrollSize   = 25
)

type model struct {
	inputs           []textinput.Model
	statusCodeView   viewport.Model
	responseTimeView viewport.Model
	responseSizeView viewport.Model
	spinner          spinner.Model
	responseView     viewport.Model
	requestHeaders   textarea.Model
	requestBody      textarea.Model
	collectionEdit   textarea.Model
	collectionView   viewport.Model
	collectionList   list.Model
	help             help.Model

	activeTab      Tab
	currentFocus   Focus
	collectionType CollectionType
	keymap         keymap

	responseViewWidth  int
	responseViewHeight int
	focusInputIndex    int
	statusCode         int
	responseTime       int64
	responseSize       int
	err                error
	startSpinner       bool
	notify             bool
	responseBody       string
	responseHeaders    string
	collectionFilePath string
	requestScheme      string
	requestHost        string
	requestBasePath    string
	requestEndpoint    string
	selectedFilter     string
	selectedItem       string
	previousItems      []string
	tabs               []string
	tabContent         []string
	collectionMap      map[string]any

	jsonRegex       *regexp.Regexp
	queryParamRegex *regexp.Regexp
}

func InitialModel(collectionFilePath string, specFile string, specVersion int) model {
	m := model{
		help:               help.New(),
		inputs:             make([]textinput.Model, 3),
		tabs:               []string{"Collection", "Request Headers", "Request Body", "Response Body", "Response Headers"},
		currentFocus:       FocusTop,
		collectionType:     CollectionList,
		spinner:            spinner.New(),
		collectionFilePath: collectionFilePath,
		keymap:             NewKeymap(),
		collectionList:     list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}
	m.collectionList.Title = "Collection"
	m.collectionList.DisableQuitKeybindings()
	m.collectionList.FilterInput.Blur()
	m.collectionList.KeyMap.NextPage.SetEnabled(false)

	var err error
	jsonRegex, err := regexp.Compile(`(\s+)"(.*)"`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong with compiling the header regex: %v", err)
		os.Exit(2)
	}
	m.jsonRegex = jsonRegex

	queryParamRegex, err := regexp.Compile(`(.*)\?(.*)=`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong with compiling the query parameter regex: %v", err)
		os.Exit(2)
	}
	m.queryParamRegex = queryParamRegex

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

		// This is to prevent the servers to be array of []any
		if collectionServers, ok := m.collectionMap["servers"].([]any); ok {
			servers := []string{}
			for _, server := range collectionServers {
				servers = append(servers, server.(string))
			}

			m.collectionMap["servers"] = servers
		}
	}

	if specFile != "" {
		servers := []string{}
		specDataStructure := map[string]map[string][]genmock.RequestStructure{}
		if specVersion == 2 {
			api, err := os.ReadFile(specFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Something went wrong with reading the spec file '%s': %v", specFile, err)
				os.Exit(2)
			}

			document, err := libopenapi.NewDocument(api)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot create new document: %e", err)
				os.Exit(2)
			}

			docModel, errors := document.BuildV2Model()

			if errors != nil {
				fmt.Fprintf(os.Stderr, "cannot build doc model: %e", errors)
				os.Exit(2)
			}

			host := strings.TrimRight(docModel.Model.Host, "/")

			for _, scheme := range docModel.Model.Schemes {
				servers = append(servers, fmt.Sprintf("%s://%s", scheme, host))
			}

			specDataStructure = genmock.SpecV2toRequestStructureMap(specFile, 1, false)
		}
		if specVersion == 3 {
			api, err := os.ReadFile(specFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Something went wrong with reading the spec file '%s': %v", specFile, err)
				os.Exit(2)
			}

			document, err := libopenapi.NewDocument(api)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot create new document: %e", err)
				os.Exit(2)
			}

			docModel, errors := document.BuildV3Model()

			if errors != nil {
				fmt.Fprintf(os.Stderr, "cannot build doc model: %e", errors)
				os.Exit(2)
			}

			specServers := docModel.Model.Servers

			for _, server := range specServers {
				servers = append(servers, strings.TrimRight(server.URL, "/"))
			}

			specDataStructure = genmock.SpecV3toRequestStructureMap(specFile, 1, false)
		}

		for method, calls := range specDataStructure {
			for path, requestStructures := range calls {
				queryParams := url.Values{}
				var body any
				for _, requestStructure := range requestStructures {
					body = requestStructure.RequestBody
					matches := m.queryParamRegex.FindStringSubmatch(requestStructure.Path)
					if len(matches) > 2 {
						queryParams[matches[2]] = []string{}
					}
				}
				m.addToCollectionMap("", "", strings.ToUpper(method), path, body, queryParams, map[string]string{})
			}
		}

		for _, server := range servers {
			if _, ok := m.collectionMap["servers"].([]string); ok {
				m.collectionMap["servers"] = append(m.collectionMap["servers"].([]string), server)
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
			t.PlaceholderStyle = placeHolderStyle
			t.Width = t.CharLimit
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		// Headers
		case 1:
			t.CharLimit = 10
			t.Placeholder = placeHolderMethod
			t.PlaceholderStyle = placeHolderStyle
			t.Width = t.CharLimit
		// jq Query
		case 2:
			t.CharLimit = 256
			t.Placeholder = placeHolderJq
			t.PlaceholderStyle = placeHolderStyle
			t.Width = t.CharLimit
		}

		m.inputs[i] = t
	}

	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinner.Moon

	m.responseView = viewport.New(78, 20)
	m.responseView.Style = windowStyle

	m.collectionView = viewport.New(78, 20)
	m.collectionView.Style = windowStyle

	m.collectionEdit = textarea.New()
	m.collectionEdit.MaxHeight = 0
	if m.collectionMap != nil {
		collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Something went wrong with parsing the file '%s': %v", m.collectionFilePath, err)
			os.Exit(2)
		}
		m.collectionEdit.SetValue(string(collectionJson))
		m.setCollectionList(m.collectionMap, "", "")
		m.currentFocus = FocusBottom
		m.inputs[0].Blur()
		m.collectionEdit.Blur()
	}
	m.collectionEdit.Cursor.Style = cursorStyle
	m.collectionEdit.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.collectionEdit.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestHeaders = textarea.New()
	m.requestHeaders.MaxHeight = 0
	m.requestHeaders.Cursor.Style = cursorStyle
	m.requestHeaders.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.requestHeaders.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestBody = textarea.New()
	m.requestBody.MaxHeight = 0
	m.requestBody.Cursor.Style = cursorStyle
	m.requestBody.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.requestBody.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.statusCodeView = viewport.New(inputWidthPadding, 1)
	m.statusCodeView.Style = statusCodeViewStyle

	m.responseTimeView = viewport.New(inputWidthPadding, 1)
	m.responseTimeView.Style = responseTimeViewStyle

	m.responseSizeView = viewport.New(inputWidthPadding, 1)
	m.responseSizeView.Style = responseSizeViewStyle

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.tabContent[TabCollection] = m.collectionEdit.Value()
	m.tabContent[TabRequestBody] = m.requestBody.Value()
	m.tabContent[TabRequestHeaders] = m.requestHeaders.Value()

	switch msg := msg.(type) {
	case responseMsg:
		m.handleResponseMsg(msg)

	case errMsg:
		m.handleErrMsg(msg)

	case spinner.TickMsg:
		if m.startSpinner {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		var err error
		cmds, err = m.handleKeyMsg(msg, cmds)
		if err != nil {
			return m, func() tea.Msg {
				return errMsg{err: err}
			}
		}
	}

	if m.currentFocus == FocusBottom && m.activeTab == TabCollection && m.collectionType == CollectionList {
		var cmd tea.Cmd
		m.collectionList, cmd = m.collectionList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder
	var renderedTabs []string

	m.updateFocusView()

	m.requestHeaders.SetWidth(m.responseViewWidth)
	m.requestHeaders.SetHeight(m.responseViewHeight)

	m.collectionEdit.SetWidth(m.responseViewWidth)
	m.collectionEdit.SetHeight(m.responseViewHeight)

	m.requestBody.SetWidth(m.responseViewWidth)
	m.requestBody.SetHeight(m.responseViewHeight)

	for i := range m.inputs {
		// weird width correction
		if m.inputs[i].Value() == "" {
			m.inputs[i].Width = m.responseViewWidth - inputWidthPadding + 3
		} else {
			m.inputs[i].Width = m.responseViewWidth - inputWidthPadding
		}
	}

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
		if m.notify && i == 0 {
			b.WriteString(m.statusCodeView.View())
		} else if !m.notify {
			if m.startSpinner && i == 0 {
				b.WriteString("    " + m.spinner.View())
			}
			if m.responseTime > 0 && i == 0 {
				b.WriteString(m.responseTimeView.View())
			}
			if m.responseSize > 0 && i == 1 {
				b.WriteString(m.responseSizeView.View())
			}
			if m.statusCode > 0 && i == 2 {
				b.WriteString(m.statusCodeView.View())
			}
		}
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	b.WriteRune('\n')
	b.WriteString(row)
	b.WriteRune('\n')
	switch m.activeTab {
	case TabCollection:
		switch m.collectionType {
		case CollectionEdit:
			b.WriteString(m.collectionEdit.View())
		case CollectionList:
			m.collectionView.SetContent(m.collectionList.View())
			b.WriteString(m.collectionView.View())
		}
	case TabRequestHeaders:
		b.WriteString(m.requestHeaders.View())
	case TabRequestBody:
		b.WriteString(m.requestBody.View())
	default:
		b.WriteString(m.responseView.View())
	}
	b.WriteRune('\n')

	b.WriteString(m.help.View(m.keymap))

	return b.String()
}

func (m *model) addToCollectionMap(scheme string, host string, method string, path string, body any, queryParameters url.Values, headers map[string]string) {
	server := scheme + "://" + host
	if m.collectionMap == nil {
		m.collectionMap = map[string]any{
			"name":     "",
			"filename": "",
			"servers":  []string{},
			"headers":  headers,
		}
	}

	if host != "" && !slices.Contains(m.collectionMap["servers"].([]string), server) {
		m.collectionMap["servers"] = append(m.collectionMap["servers"].([]string), server)
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

	for param, paramValues := range queryParameters {
		collectionMapParams := []string{}
		for _, param := range m.collectionMap[method].(map[string]any)[path].(map[string]any)[param].([]any) {
			collectionMapParams = append(collectionMapParams, param.(string))
		}
		for _, paramValue := range paramValues {
			if !slices.Contains(collectionMapParams, paramValue) {
				m.collectionMap[method].(map[string]any)[path].(map[string]any)[param] = append(m.collectionMap[method].(map[string]any)[path].(map[string]any)[param].([]any), paramValue)
			}
		}
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
	m.collectionEdit, cmds[i+2] = m.collectionEdit.Update(msg)

	return tea.Batch(cmds...)
}

func (m *model) changeFocus() {
	switch m.currentFocus {
	case FocusTop:
		m.currentFocus = FocusBottom
		m.changeActiveTab()
	case FocusBottom:
		m.currentFocus = FocusTop
		m.inputs[m.focusInputIndex].Focus()
		m.collectionEdit.Blur()
		m.requestHeaders.Blur()
		m.requestBody.Blur()
	}
}

func (m *model) changeActiveTab() {
	m.collectionEdit.Blur()
	m.requestBody.Blur()
	m.requestHeaders.Blur()
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	switch m.activeTab {
	case TabCollection:
		if m.collectionType == CollectionEdit {
			m.collectionEdit.Focus()
		}
	case TabRequestHeaders:
		m.requestHeaders.Focus()
	case TabRequestBody:
		m.requestBody.Focus()
	}
}

func (m *model) parseHeaders(curlEnvVar bool) map[string]string {
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
					if envValue != "" && !curlEnvVar {
						value = value[:start] + envValue + value[end+2:]
					}
					if curlEnvVar {
						value = value[:start] + fmt.Sprintf("${%s}", envVar) + value[end+2:]
					}
				}
			}
			headers[key] = value
		}
	}

	return headers
}

func (m *model) copyCurl() string {
	rawURL := m.inputs[0].Value()
	method := m.inputs[1].Value()
	headers := m.parseHeaders(true)
	requestBody := m.requestBody.Value()
	inputQuery := m.inputs[2].Value()
	result := fmt.Sprintf("curl -X %s", strings.ToUpper(method))

	for header, value := range headers {
		result = fmt.Sprintf("%s -H \"%s: %s\"", result, header, value)
	}

	if requestBody != "" {
		result = fmt.Sprintf("%s -d '%s'", result, strings.ReplaceAll(strings.ReplaceAll(requestBody, "\n", ""), "  ", " "))
	}

	inputUrl := rawURL
	filterQueries := []string{}
	urlSplit := strings.Split(rawURL, "?")

	if len(urlSplit) > 1 {
		inputUrl = urlSplit[0]
		filterQueries = strings.Split(urlSplit[1], "&")
		for i := range filterQueries {
			filterQueries[i] = strings.ReplaceAll(filterQueries[i], "$", "\\$")
		}
	}

	result = fmt.Sprintf("%s \"%s\"", result, inputUrl)

	if len(filterQueries) > 0 {
		result = result + " -G"
		for _, filterQuery := range filterQueries {
			result = fmt.Sprintf("%s --data-urlencode \"%s\"", result, filterQuery)
		}
	}

	if inputQuery != "" {
		result = fmt.Sprintf("%s | jq '%s'", result, inputQuery)
	}

	return result
}

func (m *model) updateFocusView() {
	switch m.currentFocus {
	case FocusBottom:
		for i := range m.inputs {
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
		windowStyle = windowStyle.BorderForeground(highlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(highlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseView.Style = windowStyle
		m.collectionView.Style = windowStyle
	case FocusTop:
		m.inputs[m.focusInputIndex].PromptStyle = focusedStyle
		m.inputs[m.focusInputIndex].TextStyle = focusedStyle
		windowStyle = windowStyle.BorderForeground(nonHighlightColor)
		inactiveTabStyle = inactiveTabStyle.BorderForeground(nonHighlightColor)
		activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
		m.responseView.Style = windowStyle
	}
}

func (m *model) setCollectionList(collectionMap map[string]any, collectionKey string, title string) {
	collectionList := []list.Item{}
	newItemSelected := true
	if collectionKey != "" {
		switch collectionMap[collectionKey].(type) {
		case map[string]string:
			for key, value := range collectionMap[collectionKey].(map[string]string) {
				collectionList = append(collectionList, getListItem(key, value))
			}
		case map[string]any:
			for key, value := range collectionMap[collectionKey].(map[string]any) {
				collectionList = append(collectionList, getListItem(key, value))
			}
		case []string:
			for _, strValue := range collectionMap[collectionKey].([]string) {
				collectionList = append(collectionList, getListItem(strValue, ""))
			}
		case []any:
			for _, value := range collectionMap[collectionKey].([]any) {
				if strValue, ok := value.(string); ok {
					collectionList = append(collectionList, getListItem(strValue, ""))
				}
			}
		default:
			newItemSelected = false
			collectionList = m.collectionList.Items()
		}
	} else {
		for key, value := range collectionMap {
			collectionList = append(collectionList, getListItem(key, value))
		}
	}
	slices.SortFunc(collectionList, func(a, b list.Item) int {
		if a.FilterValue() > b.FilterValue() {
			return 1
		} else if a.FilterValue() < b.FilterValue() {
			return -1
		}

		return 0
	})

	if len(collectionList) > 0 {
		m.collectionList.SetItems(collectionList)
	} else {
		newItemSelected = false
	}

	if title == "" {
		if collectionName, ok := collectionMap["name"].(string); ok && collectionName != "" {
			m.collectionList.Title = collectionName
		}
	} else if newItemSelected {
		m.collectionList.Title = title
	}
	if newItemSelected && m.selectedItem != collectionKey {
		m.previousItems = append(m.previousItems, m.selectedItem)
		m.selectedItem = collectionKey
	}
}

func (m *model) setRequestInputs(method, endpoint, filter, filterValue string) error {
	headers, headersOk := m.collectionMap["headers"].(map[string]any)

	if headersOk && m.requestHeaders.Value() == "" {
		newLine := ""
		for header, value := range headers {
			headerText := fmt.Sprintf("%s: %s", header, value)
			m.requestHeaders.SetValue(fmt.Sprintf("%s%s%s", m.requestHeaders.Value(), newLine, headerText))
			newLine = "\n"
		}
	}

	if method != "" {
		isWriteMethod := slices.Contains([]string{http.MethodPatch, http.MethodPost, http.MethodPut}, method)
		if m.requestHost == "" {
			var parseServer *url.URL
			var err error
			if servers, ok := m.collectionMap["servers"].([]string); ok {
				parseServer, err = url.Parse(servers[0])
			}
			if err != nil {
				return err
			}
			m.requestHost = parseServer.Host
			m.requestBasePath = parseServer.Path
			m.requestScheme = parseServer.Scheme
		}
		if !isWriteMethod && filter != "" {
			currentEndpoint := m.requestEndpoint
			matches := m.queryParamRegex.FindStringSubmatch(m.requestEndpoint)
			if len(matches) > 2 {
				currentEndpoint = matches[1]
			}
			if endpoint == currentEndpoint {
				endpoint = m.requestEndpoint
			}

			if strings.Contains(endpoint, "=") {
				endpoint += "&" + filter + "="
			} else {
				endpoint += "?" + filter + "="
			}
		}

		if !isWriteMethod && filterValue != "" && !strings.Contains(endpoint, m.selectedFilter+"="+filterValue) {
			if strings.HasSuffix(endpoint, m.selectedFilter+"=") {
				endpoint += filterValue
			} else {
				endpoint += "&" + m.selectedFilter + "=" + filterValue
			}
		}

		m.requestEndpoint = endpoint
		urlText := fmt.Sprintf("%s://%s%s%s", m.requestScheme, m.requestHost, m.requestBasePath, m.requestEndpoint)

		if isWriteMethod {
			requestBody := m.collectionMap[method].(map[string]any)[endpoint].(map[string]any)["body"]
			if requestBody != nil {
				requestBodyJson, err := json.MarshalIndent(requestBody, "", "  ")
				if err != nil {
					return err
				}
				m.requestBody.SetValue(string(requestBodyJson))
			}
		} else {
			m.requestBody.SetValue("")
		}

		m.inputs[0].SetValue(urlText)
		m.inputs[1].SetValue(method)
	}

	return nil
}

func (m *model) setResponseStatusViews() {
	// status code view
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
	padding := (m.statusCodeView.Width - len(statusMsg)) / 2
	statusCodeContent := fmt.Sprintf(" %s  %s%s", statusMsgExtra, strings.Repeat(" ", max(0, padding-5)), statusMsg)
	if len(statusCodeContent) >= inputWidthPadding {
		newStatusCodeContent := make([]byte, inputWidthPadding)
		for i := range inputWidthPadding - 4 {
			newStatusCodeContent[i] = statusCodeContent[i]
		}
		statusCodeContent = fmt.Sprintf("%s..", string(newStatusCodeContent))
	}
	m.statusCodeView.SetContent(statusCodeContent)
	m.statusCodeView.Style = statusCodeViewStyle

	// response size view
	responseSizeMsg := fmt.Sprintf("%d b", m.responseSize)
	if m.responseSize > 1000 {
		responseSizeMsg = fmt.Sprintf("%d kB", int(math.Round(float64(m.responseSize)/1000)))
	}
	if m.responseSize > 1000000 {
		responseSizeMsg = fmt.Sprintf("%d MB", int(math.Round(float64(m.responseSize)/1000000)))
	}
	paddingRespSize := (m.responseSizeView.Width - len(responseSizeMsg)) / 2
	m.responseSizeView.SetContent(fmt.Sprintf("  %s%s", strings.Repeat(" ", max(0, paddingRespSize-4)), responseSizeMsg))

	// response time view
	responseTimeMsg := fmt.Sprintf("%d ms", m.responseTime)
	if m.responseTime > 1000 {
		responseTimeMsg = fmt.Sprintf("%d s", int(math.Round(float64(m.responseTime)/1000)))
	}
	paddingRespTime := (m.responseTimeView.Width - len(responseTimeMsg)) / 2
	m.responseTimeView.SetContent(fmt.Sprintf("  %s%s", strings.Repeat(" ", max(0, paddingRespTime-4)), responseTimeMsg))
}
