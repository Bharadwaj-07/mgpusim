package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"flag"

	"runtime/debug"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/core/connections"
	"gitlab.com/yaotsu/core/engines"
	"gitlab.com/yaotsu/core/util"
	"gitlab.com/yaotsu/gcn3"
	"gitlab.com/yaotsu/gcn3/gpubuilder"
	"gitlab.com/yaotsu/gcn3/insts"
	"gitlab.com/yaotsu/gcn3/kernels"
	"gitlab.com/yaotsu/mem"
)

type hostComponent struct {
	*core.ComponentBase
}

func newHostComponent() *hostComponent {
	h := new(hostComponent)
	h.ComponentBase = core.NewComponentBase("host")
	h.AddPort("ToGpu")
	return h
}

func (h *hostComponent) Recv(req core.Req) *core.Error {
	switch req.(type) {
	case *kernels.LaunchKernelReq:
		log.Println("Kernel completed.")
	}
	return nil
}

func (h *hostComponent) Handle(evt core.Event) error {
	return nil
}

var (
	engine     core.Engine
	globalMem  *mem.IdealMemController
	gpu        *gcn3.GPU
	host       *hostComponent
	connection core.Connection
	hsaco      *insts.HsaCo
	logger     *log.Logger
)

var cpuprofile = flag.String("cpuprofile", "prof.prof", "write cpu profile to file")
var kernel = flag.String("kernel", "../disasm/kernel.hsaco", "the kernel hsaco file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.SetBlockProfileRate(1)
	go func() {
		log.Println(http.ListenAndServe("localhost:8080", nil))
	}()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		debug.PrintStack()
		os.Exit(1)
	}()

	// log.SetOutput(ioutil.Discard)
	logger = log.New(os.Stdout, "", 0)

	initPlatform()
	loadProgram()
	initMem()
	run()
	checkResult()
}

func initPlatform() {
	engine = engines.NewSerialEngine()
	engine.AcceptHook(util.NewEventLogger(log.New(os.Stdout, "", 0)))

	host = newHostComponent()
	connection = connections.NewDirectConnection(engine)

	gpuBuilder := gpubuilder.NewGPUBuilder(engine)
	gpuBuilder.Driver = host
	gpu, globalMem = gpuBuilder.BuildEmulationGPU()

	core.PlugIn(gpu, "ToDriver", connection)
	gpu.Driver = host
}

func loadProgram() {
	executable, err := elf.Open(*kernel)
	if err != nil {
		log.Fatal(err)
	}

	sec := executable.Section(".text")
	hsacoData, err := sec.Data()
	if err != nil {
		log.Fatal(err)
	}

	err = globalMem.Storage.Write(0, hsacoData)
	if err != nil {
		log.Fatal(err)
	}

	hsaco = insts.NewHsaCoFromData(hsacoData)
	fmt.Println(hsaco.Info())
}

var (
	inputHeight  = 28
	inputWidth   = 28
	maskHeight   = 5
	maskWidth    = 5
	paddedHeight = maskHeight - 1
	paddedWidth  = maskWidth - 1
)

func initMem() {
	dataStoreAddr := 4 * mem.KB
	// Write the input
	inputData := make([]byte, 0)
	buffer := bytes.NewBuffer(inputData)
	for i := 0; i < inputHeight+paddedHeight; i++ {
		for j := 0; j < inputWidth+paddedWidth; j++ {
			binary.Write(buffer, binary.LittleEndian, uint32(1))
		}
	}
	err := globalMem.Storage.Write(dataStoreAddr, buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	dataStoreAddr = dataStoreAddr + uint64((inputHeight+paddedHeight)*(inputWidth+paddedWidth)*4)
	// Write the mask
	clustersData := make([]byte, 0)
	buffer = bytes.NewBuffer(clustersData)
	for i := 0; i < maskHeight; i++ {
		for j := 0; j < maskWidth; j++ {
			binary.Write(buffer, binary.LittleEndian, float32(0.2))
		}
	}
	err = globalMem.Storage.Write(dataStoreAddr, buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}

}

func run() {
	kernelArgsBuffer := bytes.NewBuffer(make([]byte, 0))
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint64(4096))                     // input
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint64(4096+4096))                // mask
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint64(4096+4096+4096))           // output
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint32(inputHeight+paddedHeight)) // inputDimensions.x
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint32(inputWidth+paddedWidth))   // inputDimensions.y
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint32(maskHeight))               // maskDimensions.x
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint32(maskWidth))                // maskDimensions.y
	binary.Write(kernelArgsBuffer, binary.LittleEndian, uint32(paddedWidth))              // nExWidth
	err := globalMem.Storage.Write(65536, kernelArgsBuffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	req := kernels.NewLaunchKernelReq()
	req.HsaCo = hsaco
	req.Packet = new(kernels.HsaKernelDispatchPacket)
	req.Packet.GridSizeX = 2 * 64
	req.Packet.GridSizeY = 1
	req.Packet.GridSizeZ = 1
	req.Packet.WorkgroupSizeX = 64
	req.Packet.WorkgroupSizeY = 1
	req.Packet.WorkgroupSizeZ = 1
	req.Packet.KernelObject = 0
	req.Packet.KernargAddress = 65536

	var buffer bytes.Buffer
	binary.Write(&buffer, binary.LittleEndian, req.Packet)
	err = globalMem.Storage.Write(0x11000, buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	req.PacketAddress = 0x11000
	req.SetSrc(host)
	req.SetDst(gpu)
	req.SetSendTime(0)
	connErr := connection.Send(req)
	if connErr != nil {
		log.Fatal(connErr)
	}

	engine.Run()
}

func checkResult() {
	buf, err := globalMem.Storage.Read(12*mem.KB, 128*4)
	if err != nil {
		log.Fatal(nil)
	}

	for i := 0; i < inputHeight; i++ {
		for j := 0; j < inputWidth; j++ {
			bits := binary.LittleEndian.Uint32(buf[i*4 : i*4+4])
			outputs := int32(bits)
			fmt.Printf("%d, %d: %d\n", i, j, outputs)
		}
	}
}
