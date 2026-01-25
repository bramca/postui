package main

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	textInput    textinput.Model
	statusCode   int
	responseBody string
	err          error
}

func initialModel() model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 20
	ti.SetValue(placeHolderUrl)

	return model{
		textInput: ti,
	}
}

func resetModel() model {
	return model{
		err:          nil,
		statusCode:   0,
		responseBody: "",
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case responseMsg:
		m = resetModel()
		m.statusCode = msg.statusCode
		m.responseBody = msg.responseBody

		return m, nil
	case errMsg:
		m = resetModel()
		m.err = msg

		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyCtrlC.String(), "q":
			return m, tea.Quit
		case tea.KeyEnter.String():
			return m, doRequest(m.textInput.Value())
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

func (m model) View() string {
	s := m.textInput.View()

	if m.statusCode > 0 {
		s += fmt.Sprintf(" %d %s\n%s", m.statusCode, http.StatusText(m.statusCode), m.responseBody)
	}

	if m.err != nil {
		s += fmt.Sprintf("\nWe encountered an error: %v\n\n", m.err)
	}

	return s + "\n\n"
}
