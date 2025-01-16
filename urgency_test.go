package webpush

import (
	"testing"

	"github.com/i9si-sistemas/assert"
)

func TestUrgency(t *testing.T) {
	assert.True(t, isValidUrgency(UrgencyHigh))
	assert.True(t, isValidUrgency(UrgencyLow))
	assert.True(t, isValidUrgency(UrgencyNormal))
	assert.True(t, isValidUrgency(UrgencyVeryLow))
	assert.False(t, isValidUrgency(Urgency("pamonha")))
}
