package main

import (
	"encoding/json"
	"log"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
)

type Monitor struct {
	Alloc,
	TotalAlloc,
	Sys string
	Mallocs,
	Frees,
	LiveObjects,
	PauseTotalNs uint64

	NumGC        uint32
	NumGoroutine int
}

func NewMonitor(duration int) {
	var m Monitor
	var rtm runtime.MemStats
	var interval = time.Duration(duration) * time.Second
	for {
		<-time.After(interval)

		// Read full mem stats
		runtime.ReadMemStats(&rtm)

		// Number of goroutines
		m.NumGoroutine = runtime.NumGoroutine()

		// Misc memory stats
		m.Alloc = humanize.Bytes(rtm.Alloc)
		m.TotalAlloc = humanize.Bytes(rtm.TotalAlloc)
		m.Sys = humanize.Bytes(rtm.Sys)
		m.Mallocs = rtm.Mallocs
		m.Frees = rtm.Frees

		// Live objects = Mallocs - Frees
		m.LiveObjects = m.Mallocs - m.Frees

		// GC Stats
		m.PauseTotalNs = rtm.PauseTotalNs
		m.NumGC = rtm.NumGC

		// Just encode to json and print
		b, _ := json.Marshal(m)
		log.Println(string(b))
	}
}
