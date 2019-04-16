// Copyright 2019 CanonicalLtd

package call_test

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/cloud-green/sisyphus/simulation/call"
)

func TestRenderString(t *testing.T) {
	c := qt.New(t)

	attributes := call.Attributes(map[string]interface{}{
		"attr1": "test",
		"attr2": "com",
		"attr3": "username",
	})

	tests := []struct {
		str            string
		renderedString string
	}{{
		str:            "test{attr1}",
		renderedString: "testtest",
	}, {
		str:            "test{unknown}",
		renderedString: "test{unknown}",
	}, {
		str:            "http://{attr1}.{attr2}",
		renderedString: "http://test.com",
	}, {
		str:            "http://{attr1}.{attr2}/{attr3}",
		renderedString: "http://test.com/username",
	}}

	for _, test := range tests {
		str := call.RenderString(attributes, test.str)
		c.Assert(str, qt.Equals, test.renderedString)
	}
}
