package platform

import (
	"fmt"

	"gitlab.com/akita/noc"

	"gitlab.com/akita/mem/cache"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3"
	"gitlab.com/akita/gcn3/driver"
	"gitlab.com/akita/gcn3/gpubuilder"
	"gitlab.com/akita/mem"
	"gitlab.com/akita/mem/vm"
)

var UseParallelEngine bool
var DebugISA bool
var TraceInst bool
var TraceMem bool

// BuildEmuPlatform creates a simple platform for emulation purposes
func BuildEmuPlatform() (
	akita.Engine,
	*gcn3.GPU,
	*driver.Driver,
	*mem.IdealMemController,
) {
	var engine akita.Engine

	if UseParallelEngine {
		engine = akita.NewParallelEngine()
	} else {
		engine = akita.NewSerialEngine()
	}
	// engine.AcceptHook(akita.NewEventLogger(log.New(os.Stdout, "", 0)))

	mmu := vm.NewMMU("MMU", engine, &vm.DefaultPageTableFactory{})
	gpuDriver := driver.NewDriver(engine, mmu)
	connection := akita.NewDirectConnection(engine)

	gpuBuilder := gpubuilder.NewEmuGPUBuilder(engine)
	gpuBuilder.Driver = gpuDriver
	gpuBuilder.MMU = mmu
	if DebugISA {
		gpuBuilder.EnableISADebug = true
	}
	if TraceMem {
		gpuBuilder.EnableMemTracing = true
	}
	gpu, globalMem := gpuBuilder.BuildEmulationGPU()
	gpuDriver.RegisterGPU(gpu, 4*mem.GB)

	connection.PlugIn(gpuDriver.ToGPUs)
	connection.PlugIn(gpu.ToDriver)
	gpu.Driver = gpuDriver.ToGPUs

	return engine, gpu, gpuDriver, globalMem
}

//BuildR9NanoPlatform creates a platform that equips with a R9Nano GPU
func BuildR9NanoPlatform() (
	akita.Engine,
	*gcn3.GPU,
	*driver.Driver,
) {
	var engine akita.Engine

	if UseParallelEngine {
		engine = akita.NewParallelEngine()
	} else {
		engine = akita.NewSerialEngine()
	}
	//engine.AcceptHook(akita.NewEventLogger(log.New(os.Stdout, "", 0)))

	mmu := vm.NewMMU("MMU", engine, &vm.DefaultPageTableFactory{})
	gpuDriver := driver.NewDriver(engine, mmu)
	connection := akita.NewDirectConnection(engine)

	gpuBuilder := gpubuilder.R9NanoGPUBuilder{
		GPUName:           "GPU",
		Engine:            engine,
		Driver:            gpuDriver,
		EnableISADebug:    DebugISA,
		EnableMemTracing:  TraceMem,
		EnableInstTracing: TraceInst,
		MMU:               mmu,
		ExternalConn:      connection,
	}

	gpu := gpuBuilder.Build()
	gpuDriver.RegisterGPU(gpu, 4*mem.GB)

	connection.PlugIn(gpuDriver.ToGPUs)
	connection.PlugIn(gpu.ToDriver)
	connection.PlugIn(mmu.ToTop)
	gpu.Driver = gpuDriver.ToGPUs

	return engine, gpu, gpuDriver
}

//BuildNR9NanoPlatform creates a platform that equips with several R9Nano GPUs
func BuildNR9NanoPlatform(
	numGPUs int,
) (
	akita.Engine,
	*driver.Driver,
) {
	var engine akita.Engine

	if UseParallelEngine {
		engine = akita.NewParallelEngine()
	} else {
		engine = akita.NewSerialEngine()
	}
	//engine.AcceptHook(akita.NewEventLogger(log.New(os.Stdout, "", 0)))

	mmu := vm.NewMMU("MMU", engine, &vm.DefaultPageTableFactory{})
	mmu.Latency = 100
	gpuDriver := driver.NewDriver(engine, mmu)
	//connection := akita.NewDirectConnection(engine)
	connection := noc.NewFixedBandwidthConnection(32, engine, 1*akita.GHz)

	gpuBuilder := gpubuilder.R9NanoGPUBuilder{
		GPUName:           "GPU",
		Engine:            engine,
		Driver:            gpuDriver,
		EnableISADebug:    DebugISA,
		EnableMemTracing:  TraceMem,
		EnableInstTracing: TraceInst,
		MMU:               mmu,
		ExternalConn:      connection,
	}

	rdmaAddressTable := new(cache.BankedLowModuleFinder)
	rdmaAddressTable.BankSize = 4 * mem.GB
	for i := 0; i < numGPUs; i++ {
		gpuBuilder.GPUName = fmt.Sprintf("GPU_%d", i)
		gpuBuilder.GPUMemAddrOffset = uint64(i) * 4 * mem.GB
		gpu := gpuBuilder.Build()
		gpuDriver.RegisterGPU(gpu, 4*mem.GB)
		gpu.Driver = gpuDriver.ToGPUs

		gpu.RDMAEngine.RemoteRDMAAddressTable = rdmaAddressTable
		rdmaAddressTable.LowModules = append(
			rdmaAddressTable.LowModules,
			gpu.RDMAEngine.ToOutside)
		connection.PlugIn(gpu.RDMAEngine.ToOutside)
	}

	connection.PlugIn(gpuDriver.ToGPUs)
	connection.PlugIn(mmu.ToTop)

	return engine, gpuDriver
}
