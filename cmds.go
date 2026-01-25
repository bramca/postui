package main

import (
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct {
	responseBody string
	statusCode   int
}

type errMsg struct {
	err error
}

func doRequest(url string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 10 * time.Second}
		res, err := c.Get(url)

		if err != nil {
			return errMsg{err}
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return errMsg{err}
		}

		return responseMsg{
			responseBody: string(body),
			statusCode:   res.StatusCode,
		}
	}
}

func (e errMsg) Error() string {
	return e.err.Error()
}
