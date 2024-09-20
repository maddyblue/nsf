package main

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/maddyblue/nsf"
)

//go:embed html/index.html
var content embed.FS

var flagDev = flag.Bool("dev", false, "enables reading web files from disk")
var flagAddr = flag.String("addr", ":2103", "listen address")

func main() {
	flag.Parse()

	var httpfs http.FileSystem
	if *flagDev {
		httpfs = http.Dir("html")
	} else {
		sub, err := fs.Sub(content, "html")
		if err != nil {
			log.Fatal(err)
		}
		httpfs = http.FS(sub)
	}

	http.Handle("/", http.FileServer(httpfs))
	http.HandleFunc("/api/generate", Generate)

	fmt.Println("listening on", *flagAddr)
	log.Fatal(http.ListenAndServe(*flagAddr, nil))
}

func Generate(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	var data GenerateData
	if err := d.Decode(&data); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var a nsf.Apu
	a.Init()
	//DDLC NNNN
	var s1_c1 byte
	s1_c1 |= (byte(data.P1EnvDuty) & 0b11) << 6
	s1_c1 |= (bool2byte(data.P1EnvLoop)) << 5
	s1_c1 |= (bool2byte(data.P1EnvConstantVolume)) << 4
	s1_c1 |= (byte(data.P1EnvVolume) & 0b1111) << 0
	a.Write(0x00, s1_c1)
	//EPPP NSSS
	var s1_c2 byte
	s1_c2 |= (bool2byte(data.P1SweepEnable)) << 7
	s1_c2 |= (byte(data.P1SweepPeriod) & 0b111) << 4
	s1_c2 |= (bool2byte(data.P1SweepNegate)) << 3
	s1_c2 |= (byte(data.P1SweepShift) & 0b111) << 0
	a.Write(0x01, s1_c2)
	//LLLL LLLL
	var s1_c3 byte
	s1_c3 |= (byte(data.P1TimerLength)) << 0
	a.Write(0x02, s1_c3)
	//LLLL LHHH
	var s1_c4 byte
	s1_c4 |= (byte(data.P1TimerLength) & 0b11111) << 3
	s1_c4 |= (byte(data.P1TimerLength>>8) & 0b111) << 0
	a.Write(0x03, s1_c4)

	const DEFAULT_DURATION = time.Second * 10
	if data.Duration == 0 {
		data.Duration = DEFAULT_DURATION
	}
	const SAMPLE_RATE = 44100
	vols := make([]float32, 0, int(data.Duration.Seconds()*SAMPLE_RATE))
	seenNonZero := false
	firstNonZero := 0
	lastNonZero := cap(vols)
	frameTicks := 0
	sampleTicks := 0
	for sampleIdx := 0; sampleIdx < cap(vols); {
		a.Step()
		frameTicks++
		if frameTicks == nsf.CpuClock/240 {
			frameTicks = 0
			a.FrameStep()
		}
		sampleTicks++
		if sampleTicks >= nsf.CpuClock/SAMPLE_RATE {
			sampleTicks = 0
			sampleIdx++
			v := a.Volume()
			if v != 0 {
				lastNonZero = sampleIdx
				if !seenNonZero {
					seenNonZero = true
					firstNonZero = sampleIdx
				}
			} else if !seenNonZero {
				continue
			}
			vols = append(vols, v)
		}

	}
	// Trim off start and end silence.
	vols = vols[firstNonZero:lastNonZero]

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	gw := gzip.NewWriter(w)
	json.NewEncoder(gw).Encode(vols)
	gw.Close()
}

func bool2byte(b bool) byte {
	if b {
		return 0x1
	}
	return 0x0
}

type GenerateData struct {
	Duration            time.Duration `json:"duration"`
	P1EnvDuty           int           `json:"p1-env-duty"`
	P1EnvVolume         int           `json:"p1-env-volume"`
	P1SweepPeriod       int           `json:"p1-sweep-period"`
	P1SweepShift        int           `json:"p1-sweep-shift"`
	P1TimerLength       int           `json:"p1-timer-length"`
	P1TimerCounter      int           `json:"p1-timer-counter"`
	P1EnvLoop           bool          `json:"p1-env-loop"`
	P1EnvConstantVolume bool          `json:"p1-env-constant-volume"`
	P1SweepEnable       bool          `json:"p1-sweep-enable"`
	P1SweepNegate       bool          `json:"p1-sweep-negate"`
}
