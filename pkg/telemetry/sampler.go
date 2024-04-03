package telemetry

import (
	"encoding/binary"
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var _ sdktrace.Sampler = &Sampler{}

// Sampler
type Sampler struct {
	traceIDUpperBound uint64
	description       string
}

func NewSampler(fraction float64) sdktrace.Sampler {
	if fraction >= 1 {
		fraction = 1
	}
	if fraction <= 0 {
		fraction = 0
	}
	ws := &Sampler{
		traceIDUpperBound: uint64(fraction * (1 << 63)),
		description:       fmt.Sprintf("Sampler{fraction=%g}", fraction),
	}
	return ws
}

func (ws *Sampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if trace.SpanContextFromContext(p.ParentContext).IsSampled() {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}

	x := binary.BigEndian.Uint64(p.TraceID[0:8]) >> 1
	if x < ws.traceIDUpperBound {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}
	return sdktrace.SamplingResult{Decision: sdktrace.RecordOnly}
}

func (ws *Sampler) Description() string {
	return ws.description
}
