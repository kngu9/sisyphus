// Copyright 2019 CanonicalLtd

package simulation_test

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	yaml "gopkg.in/yaml.v1"

	"github.com/cloud-green/sisyphus/config"
	"github.com/cloud-green/sisyphus/simulation"
	"github.com/cloud-green/sisyphus/simulation/call"
)

var (
	simpleSim = `
constants:
  number-of-users: 1
  service-url: test.com
root-entities:
- entity: user
  cardinality: number-of-users
entities:
  user:
    initial_state: login
    attributes:
      username:
        type: random_string
        string-value: user-
        key: username
state:
  login:
    transitions:
    - state: hello-body
      probability: 1
      call:
        method: GET
        url: http://{service-url}/login
        params:
        - type: form
          attribute: username
          key: username
        results:
        - key: message
          attribute: message
  hello-body:
`
)

func TestSimulation(t *testing.T) {
	c := qt.New(t)
	callBackend := &testCallBackend{
		calls:              make(chan config.Call, 1),
		responseAttributes: make(map[string]call.Attributes),
	}

	var simConfig config.Config
	err := yaml.Unmarshal([]byte(simpleSim), &simConfig)
	c.Assert(err, qt.IsNil)

	s := simulation.New(simConfig, callBackend, 1)

	call := <-callBackend.calls
	c.Assert(call, qt.DeepEquals, config.Call{
		Method: "GET",
		URL:    "http://{service-url}/login",
		Parameters: []config.CallParameter{{
			Type:      config.FormCallParameterType,
			Attribute: "username",
			Key:       "username",
		}},
		Results: []config.CallResult{{
			Key:       "message",
			Attribute: "message",
		}},
	})
	s.Close()
}

type testCallBackend struct {
	responseAttributes map[string]call.Attributes
	responseError      error
	calls              chan config.Call
}

func (b *testCallBackend) Do(ctx context.Context, callConfig config.Call, attributes call.Attributes) (call.Attributes, error) {
	defer func() {
		b.calls <- callConfig
	}()
	if b.responseError != nil {
		return attributes, b.responseError
	}
	attrs := call.Attributes(make(map[string]interface{}, len(attributes)))
	for k, v := range attributes {
		attrs[k] = v
	}
	for k, v := range b.responseAttributes[callConfig.URL] {
		attrs[k] = v
	}
	return attrs, nil
}
