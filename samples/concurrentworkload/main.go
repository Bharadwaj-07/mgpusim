package main

import (
	"flag"
	"log"

	"net/http"
	_ "net/http/pprof"

	"gitlab.com/akita/mgpusim/benchmarks/amdappsdk/bitonicsort"
	"gitlab.com/akita/mgpusim/benchmarks/heteromark/fir"
	"gitlab.com/akita/mgpusim/samples/runner"
)

func main() {
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	runner := new(runner.Runner).ParseFlag().Init()

	firBenchmark := fir.NewBenchmark(runner.GPUDriver)
	firBenchmark.Length = 16384
	firBenchmark.SelectGPU([]int{1, 2})

	bsBenchmark := bitonicsort.NewBenchmark(runner.GPUDriver)
	bsBenchmark.Length = 1024
	bsBenchmark.SelectGPU([]int{3})

	runner.AddBenchmarkWithoutSettingGPUsToUse(firBenchmark)
	runner.AddBenchmarkWithoutSettingGPUsToUse(bsBenchmark)

	runner.Run()
}
