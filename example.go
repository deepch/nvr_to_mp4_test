package main

import (
	"log"
	"os"
	"sort"
	"strconv"

	nvr "github.com/deepch/nvr_format"
	"github.com/nareix/mp4"
)

var (
	VideoWidth  int
	VideoHeight int
)

func main() {
	quit := false
	sps := []byte{}
	pps := []byte{}
	syncCount := 0

	// rtp timestamp: 90 kHz clock rate
	// 1 sec = timestamp 90000
	timeScale := 90000

	type NALU struct {
		ts   int
		data []byte
		sync bool
	}
	var lastNALU *NALU

	var mp4w *mp4.SimpleH264Writer
	outfile, _ := os.Create("out.mp4")

	endWriteNALU := func() {
		log.Println("finish write")
		if mp4w != nil {
			if err := mp4w.Finish(); err != nil {
				panic(err)
			}
		}
	}

	writeNALU := func(sync bool, ts int, payload []byte) {
		if mp4w == nil {
			mp4w = &mp4.SimpleH264Writer{
				SPS:       sps,
				PPS:       pps,
				TimeScale: timeScale,
				W:         outfile,
				Width:     VideoWidth,
				Height:    VideoHeight,
			}
			//log.Println("SPS:\n"+hex.Dump(sps), "\nPPS:\n"+hex.Dump(pps))
		}
		curNALU := &NALU{
			ts:   ts,
			sync: sync,
			data: payload,
		}
		if lastNALU != nil {
			//log.Println("write", lastNALU.sync, len(lastNALU.data))
			if err := mp4w.WriteNALU(lastNALU.sync, curNALU.ts-lastNALU.ts, lastNALU.data); err != nil {
				panic(err)
			}
		}
		lastNALU = curNALU
	}

	handleNALU := func(nalType byte, payload []byte, ts int64) {
		if nalType == 7 {
			if len(sps) == 0 {
				sps = payload
			}
		} else if nalType == 8 {
			if len(pps) == 0 {
				pps = payload
			}
		} else if nalType == 5 {
			// keyframe
			syncCount++
			if syncCount == 5 {
				quit = true
			}
			writeNALU(true, int(ts), payload)
		} else {
			// non-keyframe
			if syncCount > 0 {
				writeNALU(false, int(ts), payload)
			}
		}
	}
	objr, _ := nvr.NewReader()
	objr.OpenFile("test.nvr")
	packet := objr.ReadTime(1448747962193884119, 1448747978310799706)
	for _, v := range sorter(packet) {
		ts, _ := strconv.ParseInt(v, 10, 64)
		handleNALU(packet[v]["frame_k"][0], packet[v]["payload"], ts/10000)
	}
	endWriteNALU()
	objr.Close()
}
func sorter(data map[string]map[string][]byte) []string {
	mk := make([]string, len(data))
	i := 0
	for k, _ := range data {
		mk[i] = k
		i++
	}
	sort.Strings(mk)
	return mk
}
