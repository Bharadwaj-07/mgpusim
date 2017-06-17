package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"gitlab.com/yaotsu/gcn3/trace/instpb"
)

var trace = make([]*instpb.Inst, 0)

func parseTrace() {
	f, err := os.Open(traceFile)
	dieOnErr(err)

	var length uint32
	for {
		err = binary.Read(f, binary.LittleEndian, &length)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Panic(err)
		}

		buf := make([]byte, length)
		n, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Panic(err)
		}
		if uint32(n) != length {
			log.Panic(errors.New("No enough bytes to load"))
		}

		instTraceItem := new(instpb.Inst)
		err = proto.Unmarshal(buf, instTraceItem)
		dieOnErr(err)

		trace = append(trace, instTraceItem)
	}

	log.Printf("%d", len(trace))
	sort.Slice(trace, func(i, j int) bool {
		return trace[i].Events[0].Time < trace[j].Events[0].Time
	})
}

func httpTrace(w http.ResponseWriter, r *http.Request) {
	start, err := strconv.ParseFloat(r.FormValue("start"), 64)
	dieOnErr(err)

	end, err := strconv.ParseFloat(r.FormValue("end"), 64)
	dieOnErr(err)

	respond := "["
	isFirst := true
	for i := 0; i < len(trace); i++ {
		inst := trace[i]

		if inst.Events[len(inst.Events)-1].Time < start {
			continue
		}

		if inst.Events[0].Time > end {
			break
		}

		bytes, err := json.Marshal(inst)
		dieOnErr(err)

		if !isFirst {
			respond += ","
		}
		isFirst = false
		respond += string(bytes)
	}
	respond += "]"

	_, err = w.Write([]byte(respond))
	dieOnErr(err)
}
