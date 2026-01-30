package main

import (
	"bytes"
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
	responseTime    int64
	statusCode      int
}

type errMsg struct {
	err error
}

func doRequest(url string, method string, headers map[string]string, requestBody string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 10 * time.Second}

		req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(requestBody)))

		for key, value := range headers {
			req.Header.Add(key, value)
		}

		if err != nil {
			return errMsg{err}
		}

		start := time.Now()
		res, err := c.Do(req)
		stop := time.Now()
		responseTime := stop.Sub(start)
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
			responseTime:    responseTime.Milliseconds(),
			statusCode:      res.StatusCode,
		}
	}
}

func (e errMsg) Error() string {
	return e.err.Error()
}
