// Copyright 2019 CanonicalLtd

package call

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/juju/errors"

	"github.com/cloud-green/sisyphus/config"
)

// HTTPClient defines an http client interface used by the http call backend.
type HTTPClient interface {
	// DoWithBody performs the specified request with specified the request body
	// and returns an http response.
	DoWithBody(req *http.Request, body io.ReadSeeker) (*http.Response, error)
}

// NewHTTPCallBackend returns a new call backend that makes http calls using
// specified call parameters.
func NewHTTPCallBackend(client HTTPClient) *httpCallBackend {
	return &httpCallBackend{
		client: client,
	}
}

type httpCallBackend struct {
	client HTTPClient
}

// Do implements the CallBackend interface.
func (c *httpCallBackend) Do(ctx context.Context, call config.Call, attributes Attributes) (Attributes, error) {
	if call.Method == "" {
		return attributes, errors.New("method not specified")
	}
	if call.URL == "" {
		return attributes, errors.New("url not specified")
	}

	resultAttributes := attributes

	url := attributes.renderString(call.URL)
	var reader io.ReadSeeker
	bodyContent := make(map[string]interface{})
	for _, p := range call.Parameters {
		if p.Type == config.BodyCallParameterType {
			bodyContent[p.Key] = attributes[p.Attribute]
		}
	}
	if len(bodyContent) > 0 {
		data, err := json.Marshal(bodyContent)
		if err != nil {
			return resultAttributes, errors.Trace(err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(call.Method, url, nil)
	if err != nil {
		return resultAttributes, errors.Trace(err)
	}
	queryValues := request.URL.Query()
	for _, p := range call.Parameters {
		switch p.Type {
		case config.FormCallParameterType:
			queryValues.Add(p.Key, fmt.Sprintf("%v", attributes[p.Attribute]))
		case config.HeaderCallParameterType:
			request.Header.Add(p.Key, fmt.Sprintf("%v", attributes[p.Attribute]))
		}

	}
	request.URL.RawQuery = queryValues.Encode()
	response, err := c.client.DoWithBody(request, reader)
	if err != nil {
		return resultAttributes, errors.Trace(err)
	}
	if response.StatusCode != http.StatusOK {
		return resultAttributes, errors.Errorf("received status code %v", response.StatusCode)
	}
	if len(call.Results) > 0 {
		if response.Body == nil {
			return resultAttributes, errors.Errorf("did no receive any response data")
		}
		decoder := json.NewDecoder(response.Body)
		values := make(map[string]string)
		if err = decoder.Decode(&values); err != nil {
			return resultAttributes, errors.Annotate(err, "failed to unmarshal response body")
		}
		for _, r := range call.Results {
			value, ok := values[r.Key]
			if !ok {
				return resultAttributes, errors.Errorf("key %q not found in the response body", r.Key)
			}
			resultAttributes[r.Attribute] = value
		}
	}
	return resultAttributes, nil
}
