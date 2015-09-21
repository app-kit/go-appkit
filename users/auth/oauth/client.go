package oauth

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	kit "github.com/theduke/go-appkit"
)

type Client interface {
	Do(method, path string, data map[string]string) (int, map[string]interface{}, kit.Error)
}

type TokenClient struct {
	service Service
	token   string
	client  *http.Client
}

// Ensure TokenClient implements Client.
var _ Client = (*TokenClient)(nil)

func NewTokenClient(service Service, token string) *TokenClient {
	return &TokenClient{
		service: service,
		token:   token,
		client:  &http.Client{},
	}
}

func (c *TokenClient) Do(method, path string, data map[string]string) (int, map[string]interface{}, kit.Error) {
	if path[0] != '/' {
		path = "/" + path
	}

	path = c.service.GetEndpointUrl() + path

	reqUrl, err := url.Parse(path)
	if err != nil {
		return 0, nil, kit.WrapError(err, "invalid_url", "")
	}

	query := reqUrl.Query()
	query.Add("access_token", c.token)

	var reader io.Reader
	var hasBodyData bool

	// Add data for GET to url query.
	if strings.ToLower(method) == "get" {
		for key := range data {
			query.Add(key, data[key])
		}
	} else {
		vals := url.Values{}
		for key := range data {
			vals.Add(key, data[key])
		}

		reader = strings.NewReader(vals.Encode())
		hasBodyData = true
	}

	reqUrl.RawQuery = query.Encode()

	req, err := http.NewRequest(method, reqUrl.String(), reader)
	if err != nil {
		return 0, nil, kit.WrapError(err, "http_request_error", "")
	}

	if hasBodyData {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, kit.WrapError(err, "http_error", "")
	}

	if resp.Body == nil {
		return resp.StatusCode, nil, nil
	}
	defer resp.Body.Close()

	rawData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, kit.WrapError(err, "body_read_error", "")
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(rawData, &respData); err != nil {
		return resp.StatusCode, nil, kit.WrapError(err, "json_unmarshal_error", "")
	}

	return resp.StatusCode, respData, nil
}
