// Copyright 2019 CanonicalLtd

package call_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/juju/errors"

	"github.com/cloud-green/sisyphus/config"
	"github.com/cloud-green/sisyphus/simulation/call"
)

func TestHTTPCallBackend(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		about              string
		config             config.Call
		attributes         call.Attributes
		responseStatus     int
		responseBody       []byte
		responseError      error
		expectedCall       httpCall
		expectedError      string
		expectedAttributes call.Attributes
	}{{
		about: "a simple GET call - everything should be ok",
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
		},
		responseStatus: http.StatusOK,
		expectedCall: httpCall{
			URL:    "/v1/test",
			Method: "GET",
			Header: http.Header{},
		},
	}, {
		about: "a GET call with query parameters",
		attributes: call.Attributes(map[string]interface{}{
			"test-attribute1": 1,
			"test-attribute2": "hello",
			"token":           "1234abc",
		}),
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
			Parameters: []config.CallParameter{{
				Type:      config.FormCallParameterType,
				Attribute: "test-attribute1",
				Key:       "username",
			}, {
				Type:      config.FormCallParameterType,
				Attribute: "test-attribute2",
				Key:       "entity",
			}, {
				Type:      config.HeaderCallParameterType,
				Attribute: "token",
				Key:       "token-key",
			}},
		},
		responseStatus: http.StatusOK,
		expectedCall: httpCall{
			URL:    "/v1/test?entity=hello&username=1",
			Method: "GET",
			Header: http.Header{
				"Token-Key": []string{"1234abc"},
			},
		},
		expectedAttributes: call.Attributes(map[string]interface{}{
			"test-attribute1": 1,
			"test-attribute2": "hello",
			"token":           "1234abc",
		}),
	}, {
		about: "a GET call response and add to attributes",
		attributes: call.Attributes(map[string]interface{}{
			"test-attribute1": 1,
		}),
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
			Results: []config.CallResult{{
				Key:       "token",
				Attribute: "authorization-token",
			}},
		},
		responseStatus: http.StatusOK,
		responseBody:   []byte(`{"token":"123456"}`),
		expectedCall: httpCall{
			URL:    "/v1/test",
			Method: "GET",
			Header: http.Header{},
		},
		expectedAttributes: call.Attributes(map[string]interface{}{
			"test-attribute1":     1,
			"authorization-token": "123456",
		}),
	}, {
		about: "a POST call with parameters and results",
		attributes: call.Attributes(map[string]interface{}{
			"test-attribute1": "test-value",
		}),
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
			Parameters: []config.CallParameter{{
				Type:      config.BodyCallParameterType,
				Attribute: "test-attribute1",
				Key:       "username",
			}},
			Results: []config.CallResult{{
				Key:       "token",
				Attribute: "authorization-token",
			}},
		},
		responseStatus: http.StatusOK,
		responseBody:   []byte(`{"token":"123456"}`),
		expectedCall: httpCall{
			URL:    "/v1/test",
			Method: "GET",
			Body:   []byte(`{"username":"test-value"}`),
			Header: http.Header{},
		},
		expectedAttributes: call.Attributes(map[string]interface{}{
			"test-attribute1":     "test-value",
			"authorization-token": "123456",
		}),
	}, {
		about: "a POST call with parameters and results - missing results",
		attributes: call.Attributes(map[string]interface{}{
			"test-attribute1": "test-value",
		}),
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
			Parameters: []config.CallParameter{{
				Type:      config.BodyCallParameterType,
				Attribute: "test-attribute1",
				Key:       "username",
			}},
			Results: []config.CallResult{{
				Key:       "token",
				Attribute: "authorization-token",
			}},
		},
		responseStatus: http.StatusOK,
		responseBody:   []byte(`{}`),
		expectedError:  `key "token" not found in the response body`,
	}, {
		about: "a POST call - does not return 200",
		attributes: call.Attributes(map[string]interface{}{
			"test-attribute1": "test-value",
		}),
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
			Parameters: []config.CallParameter{{
				Type:      config.BodyCallParameterType,
				Attribute: "test-attribute1",
				Key:       "username",
			}},
			Results: []config.CallResult{{
				Key:       "token",
				Attribute: "authorization-token",
			}},
		},
		responseStatus: http.StatusNotFound,
		expectedError:  `received status code 404`,
	}, {
		about: "a POST call - does not return 200",
		attributes: call.Attributes(map[string]interface{}{
			"test-attribute1": "test-value",
		}),
		config: config.Call{
			Method: "GET",
			URL:    "/v1/test",
			Parameters: []config.CallParameter{{
				Type:      config.BodyCallParameterType,
				Attribute: "test-attribute1",
				Key:       "username",
			}},
			Results: []config.CallResult{{
				Key:       "token",
				Attribute: "authorization-token",
			}},
		},
		responseError: errors.New("unauthorized"),
		expectedError: `unauthorized`,
	},
	}

	for i, test := range tests {
		c.Logf("running test %d: %s", i, test.about)
		client := &testHTTPClient{
			responseStatus: test.responseStatus,
			responseBody:   test.responseBody,
			responseError:  test.responseError,
		}
		backend := call.NewHTTPCallBackend(client)

		attributes, err := backend.Do(context.Background(), test.config, test.attributes)
		if test.expectedError != "" {
			c.Assert(err, qt.ErrorMatches, test.expectedError)
		} else {
			c.Assert(err, qt.IsNil)
			c.Assert(attributes, qt.DeepEquals, test.expectedAttributes)
			c.Assert(client.call, qt.DeepEquals, test.expectedCall)
		}
	}

}

type httpCall struct {
	URL    string
	Method string
	Body   []byte
	Header http.Header
}

type testHTTPClient struct {
	responseStatus int
	responseBody   []byte
	responseError  error
	call           httpCall
}

func (c *testHTTPClient) DoWithBody(req *http.Request, body io.ReadSeeker) (*http.Response, error) {
	var data []byte
	var err error
	if body != nil {
		data, err = ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}
	c.call = httpCall{
		Method: req.Method,
		URL:    req.URL.String(),
		Body:   data,
		Header: req.Header,
	}

	return &http.Response{
		StatusCode: c.responseStatus,
		Status:     http.StatusText(c.responseStatus),
		Body:       ioutil.NopCloser(bytes.NewReader(c.responseBody)),
	}, c.responseError
}
