package bodyattribute

import (
	"testing"

	"github.com/hypertrace/goagent/sdk/internal/mock"
	"github.com/stretchr/testify/assert"
)

func TestBodyTruncationSuccess(t *testing.T) {
	s := mock.NewSpan()
	SetTruncatedBodyAttribute("http.request.body", []byte("text"), 2, s)
	assert.Equal(t, "te", s.ReadAttribute("http.request.body"))
	assert.True(t, (s.ReadAttribute("http.request.body.truncated")).(bool))
	assert.Zero(t, s.RemainingAttributes())
}

func TestBodyTruncationIsSkipped(t *testing.T) {
	s := mock.NewSpan()
	SetTruncatedBodyAttribute("rpc.response.body", []byte("text"), 7, s)
	assert.Equal(t, "text", s.ReadAttribute("rpc.response.body"))
	assert.Zero(t, s.RemainingAttributes())
}

func TestBodyTruncationEmptyBody(t *testing.T) {
	s := mock.NewSpan()
	SetTruncatedBodyAttribute("body_attr", []byte{}, 7, s)
	assert.Nil(t, s.ReadAttribute("body_attr"))
	assert.Zero(t, s.RemainingAttributes())
}