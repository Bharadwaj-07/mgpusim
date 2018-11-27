package fir

import (
	"log"
	"math"

	"gitlab.com/akita/gcn3/kernels"

	"gitlab.com/akita/gcn3/driver"
	"gitlab.com/akita/gcn3/insts"
)

type KernelArgs struct {
	Output              driver.GPUPtr
	Filter              driver.GPUPtr
	Input               driver.GPUPtr
	History             driver.GPUPtr
	NumTaps             uint32
	HiddenGlobalOffsetX int64
	HiddenGlobalOffsetY int64
	HiddenGlobalOffsetZ int64
}

type Benchmark struct {
	driver *driver.Driver
	hsaco  *insts.HsaCo

	numGPUs   int
	gpuQueues []*driver.CommandQueue

	Length       int
	numTaps      int
	inputData    []float32
	filterData   []float32
	gFilterData  []driver.GPUPtr
	gHistoryData []driver.GPUPtr
	gInputData   []driver.GPUPtr
	gOutputData  []driver.GPUPtr
}

func NewBenchmark(driver *driver.Driver) *Benchmark {
	b := new(Benchmark)

	b.driver = driver
	b.numGPUs = driver.GetNumGPUs()
	for i := 0; i < b.numGPUs; i++ {
		b.driver.SelectGPU(i)
		b.gpuQueues = append(b.gpuQueues, b.driver.CreateCommandQueue())
	}

	hsacoBytes, err := Asset("kernels.hsaco")
	if err != nil {
		log.Panic(err)
	}
	b.hsaco = kernels.LoadProgramFromMemory(hsacoBytes, "FIR")

	return b
}

func (b *Benchmark) Run() {
	b.initMem()
	b.exec()
}

func (b *Benchmark) initMem() {
	b.numTaps = 16

	for i := 0; i < b.numGPUs; i++ {
		b.driver.SelectGPU(i)
		b.gFilterData = append(b.gFilterData,
			b.driver.AllocateMemory(uint64(b.numTaps*4)))
		b.gHistoryData = append(b.gFilterData,
			b.driver.AllocateMemory(uint64(b.numTaps*4)))
		b.gInputData = append(b.gInputData,
			b.driver.AllocateMemory(uint64(b.Length/b.numGPUs*4)))
		b.gOutputData = append(b.gOutputData,
			b.driver.AllocateMemory(uint64(b.Length/b.numGPUs*4)))
	}

	b.filterData = make([]float32, b.numTaps)
	for i := 0; i < b.numTaps; i++ {
		b.filterData[i] = float32(i)
	}

	b.inputData = make([]float32, b.Length)
	for i := 0; i < b.Length; i++ {
		b.inputData[i] = float32(i)
	}

	for i := 0; i < b.numGPUs; i++ {
		b.driver.EnqueueMemCopyH2D(
			b.gpuQueues[i], b.gFilterData[i], b.filterData)
		b.driver.EnqueueMemCopyH2D(
			b.gpuQueues[i], b.gInputData[i],
			b.inputData[b.Length/b.numGPUs*i:b.Length/b.numGPUs*(i+1)])
	}
}

func (b *Benchmark) exec() {
	kernArg := KernelArgs{
		b.gOutputData,
		b.gFilterData,
		b.gInputData,
		b.gHistoryData,
		uint32(b.numTaps),
		0, 0, 0,
	}

	b.driver.LaunchKernel(
		b.hsaco,
		[3]uint32{uint32(b.Length), 1, 1},
		[3]uint16{256, 1, 1},
		&kernArg,
	)
}

func (b *Benchmark) Verify() {
	gpuOutput := make([]float32, b.Length)
	b.driver.MemCopyD2H(gpuOutput, b.gOutputData)

	for i := 0; i < b.Length; i++ {
		var sum float32
		sum = 0

		for j := 0; j < b.numTaps; j++ {
			if i < j {
				continue
			}
			sum += b.inputData[i-j] * b.filterData[j]
		}

		if math.Abs(float64(sum-gpuOutput[i])) >= 1e-5 {
			log.Fatalf("At position %d, expected %f, but get %f.\n",
				i, sum, gpuOutput[i])
		}
	}

	log.Printf("Passed!\n")
}
