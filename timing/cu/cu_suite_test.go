package cu

import (
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/mgpusim/insts"
	"gitlab.com/akita/mgpusim/kernels"
)

//go:generate mockgen -write_package_comment=false -package=$GOPACKAGE -destination=mock_akita_test.go gitlab.com/akita/akita Port,Engine
//go:generate mockgen -source cucomponent.go -destination mock_cucomponent_test.go -package $GOPACKAGE
//go:generate mockgen -source wfdispatcher.go -destination mock_wfdispatcher_test.go -package $GOPACKAGE

func TestSimulator(t *testing.T) {
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "GCN3 Timing Simulator")
}

func prepareGrid(co *insts.HsaCo) *kernels.Grid {
	// Prepare a mock grid that is expanded
	grid := kernels.NewGrid()
	grid.CodeObject = co
	for i := 0; i < 5; i++ {
		wg := kernels.NewWorkGroup()
		wg.CodeObject = co
		grid.WorkGroups = append(grid.WorkGroups, wg)
		for j := 0; j < 10; j++ {
			wf := kernels.NewWavefront()
			wf.WG = wg
			wg.Wavefronts = append(wg.Wavefronts, wf)
		}
	}
	return grid
}
