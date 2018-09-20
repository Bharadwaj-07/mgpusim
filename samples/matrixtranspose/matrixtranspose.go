package main

import (
	"flag"
	"log"

	"gitlab.com/akita/akita"
	"gitlab.com/akita/gcn3"
	"gitlab.com/akita/gcn3/driver"
	"gitlab.com/akita/gcn3/insts"
	"gitlab.com/akita/gcn3/kernels"
	"gitlab.com/akita/gcn3/platform"
	"gitlab.com/akita/mem"
)

type MatrixTransposeKernelArgs struct {
	Output              driver.GPUPtr
	Input               driver.GPUPtr
	Block               driver.LocalPtr
	Padding             uint32
	HiddenGlobalOffsetX int64
	HiddenGlobalOffsetY int64
	HiddenGlobalOffsetZ int64
}

var (
	engine    akita.Engine
	globalMem *mem.IdealMemController
	storage   *mem.Storage
	gpu       *gcn3.GPU
	gpuDriver *driver.Driver
	kernel    *insts.HsaCo

	width              int
	elemsPerThread1Dim int
	blockSize          int
	hInputData         []uint32
	hOutputData        []uint32
	dInputData         driver.GPUPtr
	dOutputData        driver.GPUPtr
)

var kernelFilePath = flag.String(
	"kernel file path",
	"kernels.hsaco",
	"The path to the kernel hsaco file.",
)
var timing = flag.Bool("timing", false, "Run detailed timing simulation.")
var parallel = flag.Bool("parallel", false, "Run the simulation in parallel.")
var isaDebug = flag.Bool("debug-isa", false, "Generate the ISA debugging file.")
var instTracing = flag.Bool("trace-inst", false, "Generate instruction trace for visualization purposes.")
var verify = flag.Bool("verify", false, "Verify the emulation result.")
var dataWidth = flag.Int("width", 256, "The dimension of the square matrix.")
var memTracing = flag.Bool("trace-mem", false, "Generate memory trace")

func main() {
	configure()
	initPlatform()
	loadProgram()
	initMem()
	run()

	if *verify {
		checkResult()
	}
}

func configure() {
	flag.Parse()

	if *parallel {
		platform.UseParallelEngine = true
	}

	if *isaDebug {
		platform.DebugISA = true
	}

	if *instTracing {
		platform.TraceInst = true
	}

	if *memTracing {
		platform.TraceMem = true
	}

	width = *dataWidth
	elemsPerThread1Dim = 4
	blockSize = 16
}

func initPlatform() {
	if *timing {
		engine, gpu, gpuDriver, globalMem = platform.BuildR9NanoPlatform()
	} else {
		engine, gpu, gpuDriver, globalMem = platform.BuildEmuPlatform()
	}
	storage = globalMem.Storage
}

func loadProgram() {
	kernel = kernels.LoadProgram(*kernelFilePath, "matrixTranspose")
	if kernel == nil {
		log.Fatal("Error loading kernel")
	}
}

func initMem() {
	numData := width * width

	hInputData = make([]uint32, numData)
	hOutputData = make([]uint32, numData)

	for i := 0; i < numData; i++ {
		hInputData[i] = uint32(i)
	}

	dInputData = gpuDriver.AllocateMemory(storage, uint64(numData*4))
	dOutputData = gpuDriver.AllocateMemory(storage, uint64(numData*4))

	gpuDriver.MemoryCopyHostToDevice(dInputData, hInputData, gpu.ToDriver)
}

func run() {
	kernArg := MatrixTransposeKernelArgs{
		dOutputData,
		dInputData,
		driver.LocalPtr(blockSize * blockSize * elemsPerThread1Dim * elemsPerThread1Dim * 4),
		0,
		0, 0, 0,
	}

	gpuDriver.LaunchKernel(kernel, gpu.ToDriver, globalMem.Storage,
		[3]uint32{uint32(width / elemsPerThread1Dim), uint32(width / elemsPerThread1Dim), 1},
		[3]uint16{uint16(blockSize), uint16(blockSize), 1},
		&kernArg,
	)
}

func checkResult() {
	gpuDriver.MemoryCopyDeviceToHost(hOutputData, dOutputData, gpu.ToDriver)

	for i := 0; i < width; i++ {
		for j := 0; j < width; j++ {
			if hOutputData[j*width+i] != hInputData[i*width+j] {
				log.Printf("error at output coordinate (%d, %d), expected %d, but get %d", i, j, hInputData[i*width+j], hOutputData[j*width+i])
			}
		}
	}

	log.Printf("Passed!\n")
}
