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

var (
	highlightColor        = lipgloss.AdaptiveColor{Light: "#82aaff", Dark: "#B191FF"}
	focusedStyle          = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle           = focusedStyle
	noStyle               = lipgloss.NewStyle()
	inactiveTabBorder     = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder       = tabBorderWithBottom("┘", " ", "└")
	nonHighlightColor     = lipgloss.AdaptiveColor{Light: "#B5B5B5", Dark: "#535353"}
	inactiveTabStyle      = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor)
	activeTabStyle        = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle           = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	spinnerStyle          = lipgloss.NewStyle().Foreground(highlightColor)
	statusCodeViewStyle   = lipgloss.NewStyle().Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	responseTimeViewStyle = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "#72acff", Dark: "#c792ea"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
)

type model struct {
	inputs           []textinput.Model
	statusCodeView   viewport.Model
	responseTimeView viewport.Model
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
	selectedItem       string
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
			t.PlaceholderStyle = lipgloss.NewStyle().Foreground(nonHighlightColor)
			t.Width = t.CharLimit
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		// Headers
		case 1:
			t.CharLimit = 10
			t.Placeholder = placeHolderMethod
			t.PlaceholderStyle = lipgloss.NewStyle().Foreground(nonHighlightColor)
			t.Width = t.CharLimit
		// jq Query
		case 2:
			t.CharLimit = 256
			t.Placeholder = placeHolderJq
			t.PlaceholderStyle = lipgloss.NewStyle().Foreground(nonHighlightColor)
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
	if m.collectionMap != nil {
		collectionJson, err := json.MarshalIndent(m.collectionMap, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Something went wrong with parsing the file '%s': %v", m.collectionFilePath, err)
			os.Exit(2)
		}
		m.collectionEdit.SetValue(string(collectionJson))
		m.setCollectionList("", "")
	}
	m.collectionEdit.Cursor.Style = cursorStyle
	m.collectionEdit.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.collectionEdit.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestHeaders = textarea.New()
	m.requestHeaders.Cursor.Style = cursorStyle
	m.requestHeaders.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.requestHeaders.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.requestBody = textarea.New()
	m.requestBody.Cursor.Style = cursorStyle
	m.requestBody.BlurredStyle.Base = windowStyle.BorderForeground(nonHighlightColor)
	m.requestBody.FocusedStyle.Base = windowStyle.BorderForeground(highlightColor)

	m.statusCodeView = viewport.New(inputWidthPadding, 1)
	m.statusCodeView.Style = statusCodeViewStyle

	m.responseTimeView = viewport.New(inputWidthPadding, 1)
	m.responseTimeView.Style = responseTimeViewStyle

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
			m.statusCode = 0
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
			b.WriteString(m.responseTimeView.View())
		}
		if m.statusCode > 0 && i == 1 {
			b.WriteString(m.statusCodeView.View())
		} else if m.notify && i == 0 {
			b.WriteString(m.statusCodeView.View())
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
	m.collectionEdit, cmds[i+2] = m.collectionEdit.Update(msg)

	return tea.Batch(cmds...)
}

func (m *model) changeFocus() {
	switch m.currentFocus {
	case FocusTop:
		m.currentFocus = FocusBottom
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
	case FocusBottom:
		m.currentFocus = FocusTop
		m.inputs[m.focusInputIndex].Focus()
		m.collectionEdit.Blur()
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

func (m *model) copyCurl() string {
	rawURL := m.inputs[0].Value()
	method := m.inputs[1].Value()
	headers := m.parseHeaders()
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

func (m *model) setCollectionList(collectionKey string, title string) {
	collectionList := []list.Item{}
	newItemSelected := true
	if collectionKey != "" {
		switch m.collectionMap[collectionKey].(type) {
		case map[string]any:
			for key, value := range m.collectionMap[collectionKey].(map[string]any) {
				collectionList = append(collectionList, getListItem(key, value))
			}
		case []string:
			for _, strValue := range m.collectionMap[collectionKey].([]string) {
				collectionList = append(collectionList, getListItem(strValue, ""))
			}
		default:
			newItemSelected = false
			collectionList = m.collectionList.Items()
		}
	} else {
		for key, value := range m.collectionMap {
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

	m.collectionList.SetItems(collectionList)

	if title == "" {
		if collectionName, ok := m.collectionMap["name"].(string); ok && collectionName != "" {
			m.collectionList.Title = collectionName
		}
	} else if newItemSelected {
		m.collectionList.Title = title
		m.selectedItem = collectionKey
	}
}

func getListItem(key string, value any) item {
	listItem := item{title: key}
	switch value := value.(type) {
	case string:
		listItem = item{title: key, desc: " " + value}
	case map[string]any:
		description := ""
		for subKey := range value {
			description = fmt.Sprintf("%s %s", description, subKey)
		}
		listItem = item{title: key, desc: description}
	case []string:
		description := ""
		for _, strValue := range value {
			description = fmt.Sprintf("%s %s", description, strValue)
		}
		listItem = item{title: key, desc: description}
	}

	return listItem
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right

	return border
}
