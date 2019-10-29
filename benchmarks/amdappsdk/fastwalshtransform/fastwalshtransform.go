package fastwalshtransform

import (
	"fmt"
	"log"
	"math/rand"

	"gitlab.com/akita/gcn3/driver"
	"gitlab.com/akita/gcn3/insts"
	"gitlab.com/akita/gcn3/kernels"
)

type FastWalshTransformKernelArgs struct {
	TArray driver.GPUPtr
	Step   uint32
}

type Benchmark struct {
	driver  *driver.Driver
	context *driver.Context
	gpus    []int
	queues  []*driver.CommandQueue
	kernel  *insts.HsaCo

	Length         uint32
	hInputArray    []float32
	hVerInputArray []float32
	dInputArray    driver.GPUPtr

	useUnifiedMemory bool
}

func NewBenchmark(driver *driver.Driver) *Benchmark {
	b := new(Benchmark)
	b.driver = driver
	b.context = driver.Init()
	b.loadProgram()
	return b
}

func (b *Benchmark) SelectGPU(gpus []int) {
	b.gpus = gpus
}

func (b *Benchmark) loadProgram() {
	hsacoBytes := FSMustByte(false, "/kernels.hsaco")

	b.kernel = kernels.LoadProgramFromMemory(hsacoBytes, "fastWalshTransform")
	if b.kernel == nil {
		log.Panic("Failed to load kernel binary")
	}
}

// Use Unified Memory
func (b *Benchmark) SetUnifiedMemory() {
	b.useUnifiedMemory = true
}

func (b *Benchmark) Run() {
	for _, gpu := range b.gpus {
		b.driver.SelectGPU(b.context, gpu)
		b.queues = append(b.queues, b.driver.CreateCommandQueue(b.context))
	}

	b.initMem()
	b.exec()
	b.Verify()
}

func (b *Benchmark) initMem() {
	rand.Seed(123)

	b.hInputArray = make([]float32, b.Length)
	b.hVerInputArray = make([]float32, b.Length)

	for i := uint32(0); i < b.Length; i++ {
		temp := float32(rand.Float32() + float32(rand.Int31n(255)))
		b.hInputArray[i] = temp
		b.hVerInputArray[i] = temp
	}

	if b.useUnifiedMemory {
		b.dInputArray = b.driver.AllocateUnifiedMemory(b.context, uint64(b.Length*4))

	} else {
		b.dInputArray = b.driver.AllocateMemory(b.context, uint64(b.Length*4))
	}

	b.driver.MemCopyH2D(b.context, b.dInputArray, b.hInputArray)
}

func printArray(array []float32, n uint32) {
	for i := uint32(0); i < n; i++ {
		fmt.Printf("%f ", array[i])
	}
}

func (b *Benchmark) exec() {

	globalThreadSize := uint32(b.Length / 2)
	localThreadSize := uint16(256)

	for _, queue := range b.queues {

		for step := uint32(1); step < b.Length; step <<= 1 {

			kernArg := FastWalshTransformKernelArgs{
				TArray: b.dInputArray,
				Step:   step,
			}

			b.driver.EnqueueLaunchKernel(
				queue,
				b.kernel,
				[3]uint32{uint32(globalThreadSize), 1, 1},
				[3]uint16{uint16(localThreadSize), 1, 1},
				&kernArg,
			)
		}
	}

	for _, q := range b.queues {
		b.driver.DrainCommandQueue(q)
	}

	b.driver.MemCopyD2H(b.context, b.hInputArray, b.dInputArray)
}

func (b *Benchmark) Verify() {

	for step := uint32(1); step < b.Length; step <<= 1 {
		jump := uint32(step << 1)
		for group := uint32(0); group < step; group++ {
			for pair := uint32(group); pair < b.Length; pair += jump {
				match := uint32(pair + step)

				T1 := float32(b.hVerInputArray[pair])
				T2 := float32(b.hVerInputArray[match])

				b.hVerInputArray[pair] = T1 + T2
				b.hVerInputArray[match] = T1 - T2
			}
		}
	}

	for i := uint32(0); i < b.Length; i++ {
		if b.hInputArray[i] != b.hVerInputArray[i] {
			panic(fmt.Sprintf("Mismatch at %d, expected %f found %f",
				i, b.hInputArray[i], b.hVerInputArray[i]))
		}
	}

	log.Printf("Passed!\n")
}
