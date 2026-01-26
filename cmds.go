package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct {
	responseBody    string
	responseHeaders string
	statusCode      int
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

		headers := ""
		for header, values := range res.Header {
			headers = fmt.Sprintf("%s%s: %s\n", headers, header, strings.Join(values, ","))
		}

		return responseMsg{
			// responseBody:    fmt.Sprintf("``` json\n%s\n```", string(body)),
			// responseHeaders: fmt.Sprintf("``` yaml\n%s\n```", headers),
			responseBody:    string(body),
			responseHeaders: headers,
			statusCode:      res.StatusCode,
		}
	}
}

func (e errMsg) Error() string {
	return e.err.Error()
}
