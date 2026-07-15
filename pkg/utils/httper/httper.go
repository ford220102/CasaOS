package httper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/IceWhaleTech/CasaOS/pkg/config"
	"github.com/tidwall/gjson"
)

// sharedClient is reused across calls instead of allocating a new
// *http.Client (and its underlying transport/connection pool) per request.
var sharedClient = &http.Client{Timeout: 30 * time.Second}

// doRequest builds and executes an HTTP request, applying headers and a
// per-call timeout. It centralizes the logic that was previously
// duplicated across Get/PersonGet/Post/ZeroTierGet.
func doRequest(method, url string, body []byte, contentType string, head map[string]string, timeout time.Duration) (content string, statusCode int, err error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return "", 0, fmt.Errorf("httper: building request failed: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range head {
		req.Header.Add(k, v)
	}

	client := sharedClient
	if timeout > 0 {
		// clone timeout behavior without mutating the shared client
		client = &http.Client{Timeout: timeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("httper: request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("httper: reading response body failed: %w", err)
	}

	return string(data), resp.StatusCode, nil
}

// Get sends a GET request.
// url: request address
// head: optional request headers
// response: response body as a string (empty string on error)
func Get(url string, head map[string]string) (response string) {
	content, _, err := doRequest(http.MethodGet, url, nil, "", head, 30*time.Second)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return content
}

// PersonGet sends a GET request with a shorter (5s) timeout.
func PersonGet(url string) (response string) {
	content, _, err := doRequest(http.MethodGet, url, nil, "", nil, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return content
}

// Post sends a POST request.
// url: request address
// data: POST body
// contentType: e.g. "application/json"
// head: optional request headers
func Post(url string, data []byte, contentType string, head map[string]string) (content string) {
	result, _, err := doRequest(http.MethodPost, url, data, contentType, head, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return result
}

// ZeroTierGet sends a GET request and also returns the HTTP status code,
// since callers need to distinguish "empty body" from "request failed".
func ZeroTierGet(url string, head map[string]string) (content string, code int) {
	result, statusCode, err := doRequest(http.MethodGet, url, nil, "", head, 20*time.Second)
	if err != nil {
		fmt.Println(err)
		return "", 0
	}
	return result, statusCode
}

// OasisGet fetches an auth token and then performs a GET request with it.
// Unlike the original version, a failed token fetch is surfaced instead of
// silently proceeding with an empty Authorization header.
func OasisGet(url string) (response string) {
	tokenResp, _, err := doRequest(http.MethodGet, config.ServerInfo.ServerApi+"/token", nil, "", nil, 30*time.Second)
	if err != nil {
		fmt.Println(fmt.Errorf("httper: fetching token failed: %w", err))
		return ""
	}

	token := gjson.Get(tokenResp, "data").String()
	if token == "" {
		fmt.Println("httper: token response did not contain a 'data' field")
		return ""
	}

	head := map[string]string{"Authorization": token}
	return Get(url, head)
}
