package timingconfig

import (
	"github.com/sarchlab/akita/v4/sim"
	"github.com/sarchlab/akita/v4/simulation"
	"github.com/sarchlab/mgpusim/v4/amd/driver"
	"github.com/sarchlab/mgpusim/v4/amd/timing/tlbtracer"
)

// Builder builds a platform for timing simulation.
type Builder struct {
	simulation *simulation.Simulation
	numGPUs    int
}

// MakeBuilder creates a new builder.
func MakeBuilder() Builder {
	return Builder{}
}

// WithSimulation sets the simulation to use.
func (b Builder) WithSimulation(s *simulation.Simulation) Builder {
	b.simulation = s
	return b
}

// WithNumGPUs sets the number of GPUs to use.
func (b Builder) WithNumGPUs(n int) Builder {
	b.numGPUs = n
	return b
}

// Build builds the platform.
func (b Builder) Build() (*sim.Domain, []*tlbtracer.TLBTracer) {
	platform := sim.NewDomain("Platform")

	driverBuilder := driver.MakeBuilder().WithEngine(b.simulation.GetEngine())
	driver := driverBuilder.Build("Driver")
	b.simulation.RegisterComponent(driver)

	tracers := []*tlbtracer.TLBTracer{}
	for i := 0; i < b.numGPUs; i++ {
		tracer, err := tlbtracer.NewTLBTracer("GPU.TLBTracer", b.simulation.GetEngine())
		if err != nil {
			panic(err)
		}
		b.simulation.RegisterComponent(tracer)
		tracers = append(tracers, tracer)
	}

	return platform, tracers
}