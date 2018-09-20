// Package mock is an in-memory implementation of a compliant
// KMIP server.  It was used to develop the compliance tests,
// and can be used as a reference implementation.

package mock

import "gitlab.protectv.local/regan/kmip.git"

func NewMockServer() *MockServer {
	m := MockServer{
	}

	return &m
}

type MockServer struct {
	kmip.OperationMux
}