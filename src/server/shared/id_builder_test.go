package shared

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEllipticalTruncate(t *testing.T) {
	assert.Equal(t, "...", TruncateWithEllipsis("1 2 3", 0))
	assert.Equal(t, "1...", TruncateWithEllipsis("1 2 3", 1))
	assert.Equal(t, "1...", TruncateWithEllipsis("1 2 3", 2))
	assert.Equal(t, "1 2...", TruncateWithEllipsis("1 2 3", 3))
	assert.Equal(t, "1 2 3", TruncateWithEllipsis("1 2 3", 5))
}
