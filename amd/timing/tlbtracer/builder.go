package tlbtracer

import (
	"github.com/sarchlab/akita/v4/sim"
)

// Builder creates a new TLBTracer component
type Builder struct {
	engine sim.Engine
	name   string
}

// MakeBuilder creates a new builder
func MakeBuilder() Builder {
	return Builder{}
}

// WithEngine sets the engine
func (b Builder) WithEngine(engine sim.Engine) Builder {
	b.engine = engine
	return b
}

// WithName sets the name
func (b Builder) WithName(name string) Builder {
	b.name = name
	return b
}

// Build creates a new TLBTracer
func (b Builder) Build() *TLBTracer {
	tracer, err := NewTLBTracer(b.name, b.engine)
	if err != nil {
		panic(err)
	}
	return tracer
}
