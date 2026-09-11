package postui

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

func doRequest(rawURL string, method string, headers map[string]string, requestBody string, inputQuery string, tlsCertPath string, tlsKeyPath string, tlsCaPath string, skipTlsVerify bool) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 10 * time.Second}
		if tlsCertPath != "" && tlsKeyPath != "" {
			tlsConfig, err := buildTLSConfig(tlsCertPath, tlsKeyPath, tlsCaPath, skipTlsVerify)
			if err != nil {
				return errMsg{err}
			}
			c.Transport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
		} else if skipTlsVerify {
			c.Transport = &http.Transport{
				// #nosec: G402 // It is a delibirate feature to disable this via a command line option
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}

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
			responseBodyContent, err = parseResponseBody(responseBodyContent, inputQuery)
			if err != nil {
				return errMsg{err}
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

func buildTLSConfig(certPath string, keyPath string, caPath string, insecureSkipVerify bool) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// #nosec: G402 // It is a delibirate feature to disable this via a command line option
		InsecureSkipVerify: insecureSkipVerify,
	}

	certPath = strings.TrimSpace(certPath)
	keyPath = strings.TrimSpace(keyPath)
	caPath = strings.TrimSpace(caPath)

	if certPath != "" || keyPath != "" {
		if certPath == "" || keyPath == "" {
			return nil, fmt.Errorf("both TLS certificate and key paths must be set together")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed loading TLS key pair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if caPath != "" {
		// #nosec G304 -- path is supplied via trusted operator config/secret mount.
		caCert, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("failed reading TLS CA path: %w", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed parsing CA certificate in TLS CA path")
		}
		cfg.RootCAs = caPool
	}

	return cfg, nil
}

func parseResponseBody(body string, inputQuery string) (string, error) {
	query, err := gojq.Parse(inputQuery)
	if err != nil {
		return "", err
	}

	var jsonBody any
	err = json.Unmarshal([]byte(body), &jsonBody)
	if err != nil {
		return "", err
	}

	iter := query.Run(jsonBody)
	responseBodyContent := ""
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
				return "", err
			}
		}

		switch t := v.(type) {
		case map[string]any:
			jsonResult, err := json.MarshalIndent(t, "", "  ")
			if err != nil {
				return "", err
			}

			responseBodyContent = fmt.Sprintf("%s%s\n", responseBodyContent, jsonResult)
		case []any:
			jsonResult, err := json.MarshalIndent(t, "", "  ")
			if err != nil {
				return "", err
			}

			responseBodyContent = fmt.Sprintf("%s%s\n", responseBodyContent, jsonResult)
		default:
			responseBodyContent = fmt.Sprintf("%s%#v\n", responseBodyContent, t)
		}
	}

	return responseBodyContent, nil
}

func (e errMsg) Error() string {
	return e.err.Error()
}
