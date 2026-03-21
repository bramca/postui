package postui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/itchyny/gojq"
)

type responseMsg struct {
	responseBody    string
	responseHeaders string
	responseTime    int64
	responseSize    int
	statusCode      int
}

type errMsg struct {
	err error
}

func doRequest(rawURL string, method string, headers map[string]string, requestBody string, inputQuery string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 10 * time.Second}

		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to parse URL: %v", err)}
		}

		// Encode query parameters if they exist
		parsedURL.RawQuery = parsedURL.Query().Encode()

		req, err := http.NewRequest(method, parsedURL.String(), bytes.NewBuffer([]byte(requestBody)))

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

		responseBodyContent := string(body)
		if inputQuery != "" {
			query, err := gojq.Parse(inputQuery)
			if err != nil {
				return errMsg{err: err}
			}

			var jsonBody any
			err = json.Unmarshal([]byte(body), &jsonBody)
			if err != nil {
				return errMsg{err: err}
			}

			iter := query.Run(jsonBody)
			responseBodyContent = ""
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
						break
					}

					if err != nil {
						return errMsg{err: err}
					}
				}

				switch t := v.(type) {
				case map[string]any:
					jsonResult, err := json.MarshalIndent(t, "", "  ")
					if err != nil {
						return errMsg{err: err}
					}

					responseBodyContent = fmt.Sprintf("%s%s\n", responseBodyContent, jsonResult)
				case []any:
					jsonResult, err := json.MarshalIndent(t, "", "  ")
					if err != nil {
						return errMsg{err: err}
					}

					responseBodyContent = fmt.Sprintf("%s%s\n", responseBodyContent, jsonResult)
				default:
					responseBodyContent = fmt.Sprintf("%s%#v\n", responseBodyContent, t)
				}
			}

		} else {
			var prettyJson bytes.Buffer
			err := json.Indent(&prettyJson, body, "", "  ")
			if err == nil {
				responseBodyContent = prettyJson.String()
			}
		}

		return responseMsg{
			responseBody:    responseBodyContent,
			responseHeaders: headers,
			responseTime:    responseTime.Milliseconds(),
			responseSize:    len(body),
			statusCode:      res.StatusCode,
		}
	}
}

func (e errMsg) Error() string {
	return e.err.Error()
}
