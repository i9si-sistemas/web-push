package webpush

import (
	"context"
	"testing"

	"github.com/i9si-sistemas/assert"
	"github.com/i9si-sistemas/nine"
)

func TestClient(t *testing.T) {
	assert.NotNil(t, New(nine.New(context.Background())))
}
