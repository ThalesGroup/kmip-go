package ttlv_test

import (
	"github.com/gemalto/kmip-go/kmip14"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTag_CanonicalName(t *testing.T) {
	assert.Equal(t, "Cryptographic Algorithm", kmip14.TagCryptographicAlgorithm.CanonicalName())
}
