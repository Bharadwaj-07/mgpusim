package bitonicsort

import (
	"log"
	"math/rand"

	"gitlab.com/akita/gcn3/driver"
	"gitlab.com/akita/gcn3/insts"
	"gitlab.com/akita/gcn3/kernels"
)

type BitonicKernelArgs struct {
	Input               driver.GPUPtr
	Stage               uint32
	PassOfStage         uint32
	Direction           uint32
	HiddenGlobalOffsetX int64
	HiddenGlobalOffsetY int64
	HiddenGlobalOffsetZ int64
}

type Benchmark struct {
	driver *driver.Driver

	hsaco *insts.HsaCo

	Length         int
	OrderAscending bool

	inputData  []uint32
	gInputData driver.GPUPtr
}

func NewBenchmark(driver *driver.Driver) *Benchmark {
	b := new(Benchmark)
	b.driver = driver
	b.loadProgram()
	return b
}

func (b *Benchmark) loadProgram() {
	hsacoBytes, err := Asset("kernels.hsaco")
	if err != nil {
		log.Panic(err)
	}

	b.hsaco = kernels.LoadProgramFromMemory(hsacoBytes, "BitonicSort")
	if b.hsaco == nil {
		log.Panic("Failed to load kernel binary")
	}
}

func (b *Benchmark) Run() {
	b.initMem()
	b.exec()
}

func (b *Benchmark) initMem() {
	b.gInputData = b.driver.AllocateMemory(uint64(b.Length * 4))

	b.inputData = make([]uint32, b.Length)
	for i := 0; i < b.Length; i++ {
		b.inputData[i] = rand.Uint32()
	}

	b.driver.MemoryCopyHostToDevice(b.gInputData, b.inputData)
}

func (b *Benchmark) exec() {

	numStages := 0
	for temp := b.Length; temp > 1; temp >>= 1 {
		numStages++
	}

	direction := 1
	if b.OrderAscending == false {
		direction = 0
	}

	for stage := 0; stage < numStages; stage += 1 {
		for passOfStage := 0; passOfStage < stage+1; passOfStage++ {
			kernArg := BitonicKernelArgs{
				b.gInputData,
				uint32(stage),
				uint32(passOfStage),
				uint32(direction),
				0, 0, 0}
			b.driver.LaunchKernel(b.hsaco,
				[3]uint32{uint32(b.Length / 2), 1, 1},
				[3]uint16{256, 1, 1},
				&kernArg)
		}

	}
}

func (b *Benchmark) Verify() {
	gpuOutput := make([]uint32, b.Length)
	b.driver.MemoryCopyDeviceToHost(gpuOutput, b.gInputData)

	for i := 0; i < b.Length-1; i++ {
		if b.OrderAscending {
			if gpuOutput[i] > gpuOutput[i+1] {
				log.Fatalf("Error: array[%d] > array[%d]: %d %d\n", i, i+1,
					gpuOutput[i], gpuOutput[i+1])
			}
		} else {
			if gpuOutput[i] < gpuOutput[i+1] {
				log.Fatalf("Error: array[%d] < array[%d]: %d %d\n", i, i+1,
					gpuOutput[i], gpuOutput[i+1])
			}
		}
	}

	log.Printf("Passed!\n")
}
