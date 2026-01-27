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

func doRequest(url string, method string, body string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 10 * time.Second}
		var bodyReader io.Reader

		if body != "" {
			bodyReader = strings.NewReader(body)
		}

		req, err := http.NewRequest(method, url, bodyReader)
		time.Sleep(5 * time.Second)
		if err != nil {
			return errMsg{err}
		}

		res, err := c.Do(req)
		if err != nil {
			return errMsg{err}
		}

		defer func() {
			err = res.Body.Close()
		}()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return errMsg{err}
		}

		headers := ""
		for header, values := range res.Header {
			headers = fmt.Sprintf("%s%s: %s\n", headers, header, strings.Join(values, ","))
		}

		return responseMsg{
			responseBody:    string(body),
			responseHeaders: headers,
			statusCode:      res.StatusCode,
		}
	}
}

func (e errMsg) Error() string {
	return e.err.Error()
}
