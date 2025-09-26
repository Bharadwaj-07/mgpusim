package tlbtracer

import (
	"container/list"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sarchlab/akita/v4/mem/vm"
	"github.com/sarchlab/akita/v4/sim"
	"gopkg.in/yaml.v2"
)

// TLBRequest is a wrapper for a translation request that is being tracked.
type TLBRequest struct {
	Req         *vm.TranslationReq
	RequestTime sim.VTimeInSec
}

// SummaryStats holds the aggregated statistics for the TLB tracer.
type SummaryStats struct {
	TotalRequests uint64 `yaml:"total_requests"`
	Outcomes      struct {
		L1VHits    uint64 `yaml:"l1v_hits"`
		L2Hits     uint64 `yaml:"l2_hits"`
		PageWalks  uint64 `yaml:"page_walks"`
		PageFaults uint64 `yaml:"page_faults"`
	} `yaml:"outcomes"`
	HitRatesPercent struct {
		L1VHitRate     float64 `yaml:"l1v_hit_rate"`
		OverallHitRate float64 `yaml:"overall_hit_rate"`
	} `yaml:"hit_rates_percent"`
	MissRatePercent float64 `yaml:"miss_rate_percent"`
	MissTypeCounts  struct {
		Compulsory uint64 `yaml:"compulsory"`
		Capacity   uint64 `yaml:"capacity"`
		Conflict   uint64 `yaml:"conflict"`
	} `yaml:"miss_type_counts"`
	AverageLatencies struct {
		L1VHit    float64 `yaml:"l1v_hit"`
		L2Hit     float64 `yaml:"l2_hit"`
		PageWalk  float64 `yaml:"page_walk"`
		PageFault float64 `yaml:"page_fault"`
	} `yaml:"average_latencies"`
	TotalEvictions          uint64         `yaml:"total_evictions"`
	PerPageEvictionCounts map[string]int `yaml:"per_page_eviction_counts"`

	// Internal sums for calculating averages
	totalL1VHitLatency    sim.VTimeInSec
	totalL2HitLatency     sim.VTimeInSec
	totalPageWalkLatency  sim.VTimeInSec
	totalPageFaultLatency sim.VTimeInSec
}

// TLBTracer is a component that traces TLB requests and responses.
type TLBTracer struct {
	*sim.TickingComponent
	TopPort     sim.Port
	BottomPort  sim.Port
	ControlPort sim.Port
	csvWriter   *csv.Writer
	file        *os.File

	pendingRequests map[string]*TLBRequest

	// Shadow TLB for eviction tracking
	tlbSize    int
	tlbWays    int
	tlbSets    int
	tlbData    map[int]*list.List // Map of setID -> LRU list of VAddrs
	tlbEntries map[uint64]*list.Element
	lruList    *list.List // Used for fully associative caches

	// Stats
	stats          *SummaryStats
	evictionCounts map[uint64]int
	seenPages      map[uint64]bool
}

// NewTLBTracer creates a new TLBTracer.
func NewTLBTracer(name string, engine sim.Engine) (*TLBTracer, error) {
	t := new(TLBTracer)
	t.TickingComponent = sim.NewTickingComponent(name, engine, 1*sim.GHz, t)

	t.TopPort = sim.NewPort(t, 1, 1, name+".TopPort")
	t.BottomPort = sim.NewPort(t, 1, 1, name+".BottomPort")
	t.ControlPort = sim.NewPort(t, 1, 1, name+".ControlPort")

	t.pendingRequests = make(map[string]*TLBRequest)

	// Hardcoded based on shaderarray/builder.go
	t.tlbSize = 64
	t.tlbSets = 1
	t.tlbWays = 64

	t.tlbData = make(map[int]*list.List)
	for i := 0; i < t.tlbSets; i++ {
		t.tlbData[i] = list.New()
	}
	t.tlbEntries = make(map[uint64]*list.Element)
	t.lruList = list.New()

	t.stats = new(SummaryStats)
	t.stats.PerPageEvictionCounts = make(map[string]int)
	t.evictionCounts = make(map[uint64]int)
	t.seenPages = make(map[uint64]bool)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(wd, "tlb_trace.csv")
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	t.file = file

	t.csvWriter = csv.NewWriter(file)
	t.csvWriter.Write([]string{
		"Timestamp", "Type", "PID", "VAddr", "Latency",
		"Outcome", "EvictedVAddr", "MissType", "EvictedAddrTotalEvictions",
	})
	t.csvWriter.Flush()

	return t, nil
}

// Tick processes events.
func (t *TLBTracer) Tick() bool {
	madeProgress := false
	madeProgress = t.processFromTop() || madeProgress
	madeProgress = t.processFromBottom() || madeProgress
	return madeProgress
}

func (t *TLBTracer) processFromTop() bool {
	req := t.TopPort.PeekIncoming()
	if req == nil {
		return false
	}

	switch req := req.(type) {
	case *vm.TranslationReq:
		t.stats.TotalRequests++
		t.pendingRequests[req.ID] = &TLBRequest{
			Req:         req,
			RequestTime: t.Engine.CurrentTime(),
		}
	}

	err := t.BottomPort.Send(req)
	if err == nil {
		t.TopPort.RetrieveIncoming()
		return true
	}

	return false
}

func (t *TLBTracer) processFromBottom() bool {
	rsp := t.BottomPort.PeekIncoming()
	if rsp == nil {
		return false
	}

	switch rsp := rsp.(type) {
	case *vm.TranslationRsp:
		reqInfo, ok := t.pendingRequests[rsp.GetRspTo()]
		if !ok {
			break
		}
		delete(t.pendingRequests, rsp.GetRspTo())

		t.logResponse(reqInfo, rsp)
	}

	err := t.TopPort.Send(rsp)
	if err == nil {
		t.BottomPort.RetrieveIncoming()
		return true
	}

	return false
}

func (t *TLBTracer) logResponse(reqInfo *TLBRequest, rsp *vm.TranslationRsp) {
	latency := t.Engine.CurrentTime() - reqInfo.RequestTime
	vAddr := reqInfo.Req.VAddr
	pid := reqInfo.Req.PID
	pageSize := uint64(1 << 12) // Assuming 4KB pages
	vPage := vAddr / pageSize

	// Determine Outcome
	outcome := "PAGE_WALK"
	if rsp.Page.IsMigrating { // Using IsMigrating as a proxy for a page fault
		outcome = "PAGE_FAULT"
		t.stats.Outcomes.PageFaults++
		t.stats.totalPageFaultLatency += latency
	} else if latency <= 4 {
		outcome = "L1V_HIT"
		t.stats.Outcomes.L1VHits++
		t.stats.totalL1VHitLatency += latency
	} else if latency <= 60 {
		outcome = "L2_HIT"
		t.stats.Outcomes.L2Hits++
		t.stats.totalL2HitLatency += latency
	} else {
		t.stats.Outcomes.PageWalks++
		t.stats.totalPageWalkLatency += latency
	}

	// Simulate Shadow TLB
	evictedVAddrStr := ""
	missType := ""
	evictedAddrCountStr := ""

	_, pageSeen := t.seenPages[vPage]
	t.seenPages[vPage] = true

	_, isHit := t.tlbEntries[vPage]
	if isHit {
		t.lruList.MoveToFront(t.tlbEntries[vPage])
	} else { // A miss occurred
		if !pageSeen {
			missType = "COMPULSORY"
			t.stats.MissTypeCounts.Compulsory++
		} else {
			// Since the TLB is fully associative, any non-compulsory miss is a capacity miss.
			missType = "CAPACITY"
			t.stats.MissTypeCounts.Capacity++
		}

		if t.lruList.Len() >= t.tlbSize {
			// Eviction
			lruElement := t.lruList.Back()
			evictedVPage := lruElement.Value.(uint64)
			evictedVAddr := evictedVPage * pageSize

			delete(t.tlbEntries, evictedVPage)
			t.lruList.Remove(lruElement)

			t.stats.TotalEvictions++
			t.evictionCounts[evictedVAddr]++
			evictedVAddrStr = fmt.Sprintf("0x%x", evictedVAddr)
			evictedAddrCountStr = fmt.Sprintf("%d", t.evictionCounts[evictedVAddr])
		}

		newElement := t.lruList.PushFront(vPage)
		t.tlbEntries[vPage] = newElement
	}

	// Write to CSV
	t.csvWriter.Write([]string{
		fmt.Sprintf("%.10f", t.Engine.CurrentTime()),
		"Response",
		fmt.Sprintf("%d", pid),
		fmt.Sprintf("0x%x", vAddr),
		fmt.Sprintf("%.0f", float64(latency)),
		outcome,
		evictedVAddrStr,
		missType,
		evictedAddrCountStr,
	})
	t.csvWriter.Flush()
}

// Finalize calculates and writes the summary statistics.
func (t *TLBTracer) Finalize() {
	t.csvWriter.Flush()
	err := t.file.Close()
	if err != nil {
		log.Printf("Error closing tlb_trace.csv: %v", err)
	}

	// Calculate final stats
	if t.stats.TotalRequests > 0 {
		s := t.stats
		s.HitRatesPercent.L1VHitRate = float64(s.Outcomes.L1VHits) / float64(s.TotalRequests) * 100
		s.HitRatesPercent.OverallHitRate = float64(s.Outcomes.L1VHits+s.Outcomes.L2Hits) / float64(s.TotalRequests) * 100
		s.MissRatePercent = float64(s.Outcomes.PageWalks+s.Outcomes.PageFaults) / float64(s.TotalRequests) * 100
	}

	if s := t.stats.Outcomes; s.L1VHits > 0 {
		t.stats.AverageLatencies.L1VHit = float64(t.stats.totalL1VHitLatency) / float64(s.L1VHits)
	}
	if s := t.stats.Outcomes; s.L2Hits > 0 {
		t.stats.AverageLatencies.L2Hit = float64(t.stats.totalL2HitLatency) / float64(s.L2Hits)
	}
	if s := t.stats.Outcomes; s.PageWalks > 0 {
		t.stats.AverageLatencies.PageWalk = float64(t.stats.totalPageWalkLatency) / float64(s.PageWalks)
	}
	if s := t.stats.Outcomes; s.PageFaults > 0 {
		t.stats.AverageLatencies.PageFault = float64(t.stats.totalPageFaultLatency) / float64(s.PageFaults)
	}

	for vAddr, count := range t.evictionCounts {
		t.stats.PerPageEvictionCounts[fmt.Sprintf("0x%x", vAddr)] = count
	}

	// Write summary file
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory for summary: %v", err)
		return
	}
	summaryFilePath := filepath.Join(wd, "tlb_summary_stats.yaml")
	yamlData, err := yaml.Marshal(t.stats)
	if err != nil {
		log.Printf("Error marshalling summary stats: %v", err)
		return
	}

	err = os.WriteFile(summaryFilePath, yamlData, 0644)
	if err != nil {
		log.Printf("Error writing summary stats file: %v", err)
	}
}