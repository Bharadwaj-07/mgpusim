package driver

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"encoding/binary"

	"gitlab.com/yaotsu/core"
	"gitlab.com/yaotsu/gcn3"
	"gitlab.com/yaotsu/gcn3/insts"
)

// A Driver of the GCN3Sim is a Yaotsu component that receives requests from
// the runtime and directly controls the simulator.
type Driver struct {
	*core.ComponentBase

	GPUs []*gcn3.GPU
}

// NewDriver returns a newly created driver.
func NewDriver(name string) *Driver {
	d := new(Driver)
	d.ComponentBase = core.NewComponentBase(name)

	d.GPUs = make([]*gcn3.GPU, 0)

	return d
}

// Listen wait for the clients to connect in.
func (d *Driver) Listen() {
	l, err := net.Listen("tcp", "localhost:13000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go d.handleConnection(conn)
	}
}

func (d *Driver) handleConnection(conn net.Conn) {
	for {
		numberBuf := make([]byte, 8)
		lengthBuf := make([]byte, 8)
		var args []byte

		if _, err := io.ReadFull(conn, numberBuf); err != nil {
			log.Print(err)
			break
		}
		number := insts.BytesToUint64(numberBuf)

		if _, err := io.ReadFull(conn, lengthBuf); err != nil {
			log.Print(err)
			break
		}
		length := insts.BytesToUint64(lengthBuf)

		if length > 0 {
			args = make([]byte, length)
			if _, err := io.ReadFull(conn, args); err != nil {
				log.Print(err)
				break
			}
		}

		d.handleIOCTL(number, args, conn)
	}
}

func (d *Driver) handleIOCTL(number uint64, args []byte, conn net.Conn) {
	switch number {
	case 0x01:
		d.handleIOCTLGetVersion(conn)
	case 0x05:
		d.handleIOCTLGetClockCounters(args, conn)
	case 0x21:
		d.handleIOCTLAcquireSystemProperties(conn)
	case 0x22:
		d.handleIOCTLGetNodeProperties(args, conn)
	case 0x23:
		d.handleIOCTLGetNodeMemProperties(args, conn)
	default:
		log.Printf("IOCTL number 0x%02x is not supported.", number)
	}
}

type kfdIOCTLGetVersionArgs struct {
	majorVersion, minorVersion uint32
}

// IOCTL 0x01
func (d *Driver) handleIOCTLGetVersion(conn net.Conn) {
	args := new(kfdIOCTLGetVersionArgs)
	args.majorVersion = 1
	args.minorVersion = 0

	binary.Write(conn, binary.LittleEndian, args)
}

type kfdIOCTLGetClockCounters struct {
	gpuClockCounter, cpuClockCounter, systemClockCounter, systemClockFreq uint64
	nodeID, pad                                                           uint32
}

// IOCTL 0x05
func (d *Driver) handleIOCTLGetClockCounters(args []byte, conn net.Conn) {
	prop := new(kfdIOCTLGetClockCounters)
	binary.Read(bytes.NewReader(args), binary.LittleEndian, &prop)

	node := d.GPUs[prop.nodeID]
	prop.systemClockFreq = uint64(node.Freq)

	binary.Write(conn, binary.LittleEndian, prop)
}

type kfdIOCTLAcquireSystemProperties struct {
	numNodes uint32
}

// IOCTL 0x21
func (d *Driver) handleIOCTLAcquireSystemProperties(conn net.Conn) {
	args := new(kfdIOCTLAcquireSystemProperties)
	args.numNodes = uint32(len(d.GPUs))

	binary.Write(conn, binary.LittleEndian, args)
}

type kfdIOCTLGetNodeProperties struct {
	nodeID      uint32
	numCU       uint32
	engineID    uint32
	numMemBanks uint32
}

// IOCTL 0x22
func (d *Driver) handleIOCTLGetNodeProperties(
	args []byte,
	conn net.Conn,
) {
	prop := new(kfdIOCTLGetNodeProperties)
	binary.Read(bytes.NewReader(args), binary.LittleEndian, &prop)

	node := d.GPUs[prop.nodeID]
	prop.numCU = uint32(len(node.CUs))
	prop.engineID = 3<<24 + 0<<16 + 8<<10 // GFX 803
	prop.numMemBanks = 1

	binary.Write(conn, binary.LittleEndian, prop)
}

type kfdIOCTLGetNodeMemoryProperties struct {
	nodeID             uint32
	bankID             uint32
	heapType           uint32
	flags              uint32
	width              uint32
	maxClockMHz        uint32
	byteSize           uint64
	virtualBaseAddress uint64
}

func (d *Driver) handleIOCTLGetNodeMemProperties(
	args []byte,
	conn net.Conn,
) {
	prop := new(kfdIOCTLGetNodeMemoryProperties)
	binary.Read(bytes.NewReader(args), binary.LittleEndian, &prop)

	// node := d.GPUs[prop.nodeID]
	prop.heapType = 2 // Private
	prop.flags = 0
	prop.width = 512
	prop.maxClockMHz = 500
	prop.byteSize = 4 << 30 // 4 GB
	prop.virtualBaseAddress = 0

	binary.Write(conn, binary.LittleEndian, prop)
}
