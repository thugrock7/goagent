package mock

import (
	"context"
	"sync"
	"time"

	"github.com/hypertrace/goagent/sdk"
)

type spanEvent struct {
	name       string
	ts         time.Time
	attributes map[string]interface{}
}

type Status struct {
	Code    sdk.Code
	Message string
}

var _ sdk.AttributeList = (*AttributeList)(nil)

type AttributeList struct {
	attrs map[string]interface{}
}

func (l *AttributeList) GetValue(key string) interface{} {
	return l.attrs[key]
}

func (l *AttributeList) Iterate(yield func(key string, value interface{}) bool) {
	for key, value := range l.attrs {
		if !yield(key, value) {
			return
		}
	}
}

func (l *AttributeList) Len() int {
	return len(l.attrs)
}

var _ sdk.Span = &Span{}

type Span struct {
	Name       string
	Attributes map[string]interface{}
	Options    sdk.SpanOptions
	Err        error
	Noop       bool
	Status     Status
	spanEvents []spanEvent
	mux        *sync.Mutex
}

func NewSpan() *Span {
	return &Span{mux: &sync.Mutex{}}
}

func (s *Span) GetAttributes() sdk.AttributeList {
	return &AttributeList{
		attrs: s.Attributes,
	}
}

func (s *Span) SetAttribute(key string, value interface{}) {
	s.mux.Lock() // avoids race conditions
	defer s.mux.Unlock()

	if s.Attributes == nil {
		s.Attributes = make(map[string]interface{})
	}
	s.Attributes[key] = value
}

func (s *Span) ReadAttribute(key string) interface{} {
	s.mux.Lock() // avoids race conditions
	defer s.mux.Unlock()

	val, ok := s.Attributes[key]
	if ok {
		delete(s.Attributes, key)
		return val
	}

	return nil
}

func (s *Span) RemainingAttributes() int {
	return len(s.Attributes)
}

func (s *Span) SetStatus(code sdk.Code, description string) {
	s.Status = Status{
		Code:    code,
		Message: description,
	}
}

func (s *Span) SetError(err error) {
	s.Err = err
}

func (s *Span) IsNoop() bool {
	return s.Noop
}

func (s *Span) AddEvent(name string, ts time.Time, attributes map[string]interface{}) {
	s.mux.Lock() // avoids race conditions
	defer s.mux.Unlock()

	s.spanEvents = append(s.spanEvents, spanEvent{name, ts, attributes})
}

// This function has no use, it has been added just so that the interface in sdk/span.go remains implemented
func (s *Span) GetSpanId() string {
	return ""
}

type spanKey string

func SpanFromContext(ctx context.Context) sdk.Span {
	return ctx.Value(spanKey("span")).(*Span)
}

func StartSpan(ctx context.Context, name string, opts *sdk.SpanOptions) (context.Context, sdk.Span, func()) {
	s := &Span{Name: name, Options: *opts}
	return ContextWithSpan(ctx, s), s, func() {}
}

func ContextWithSpan(ctx context.Context, s sdk.Span) context.Context {
	return context.WithValue(ctx, spanKey("span"), s)
}
