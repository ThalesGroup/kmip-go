package ttlv_test

import (
	"testing"

	"github.com/gemalto/kmip-go/kmip14"
	"github.com/stretchr/testify/assert"
)

func TestTag_CanonicalName(t *testing.T) {
	assert.Equal(t, "Cryptographic Algorithm", kmip14.TagCryptographicAlgorithm.CanonicalName())
}
