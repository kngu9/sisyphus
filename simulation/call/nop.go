// Copyright 2019 CanonicalLtd

package call

import (
	"context"

	"github.com/cloud-green/sisyphus/config"
)

// New NOPCallBackend returns a call backend that does not do anything
// when executing a call.
func NewNOPCallBackend() *nopCallBackend {
	return &nopCallBackend{}
}

type nopCallBackend struct{}

// Do implements the CallBackend interface.
func (c *nopCallBackend) Do(ctx context.Context, call config.Call, attributes Attributes) (Attributes, error) {
	return attributes, nil
}
