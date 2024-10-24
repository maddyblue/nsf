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
	"os"
	"time"

	"github.com/maddyblue/nsf"
)

//go:embed html static
var content embed.FS

func contentFS(path string) http.Handler {
	var httpfs http.FileSystem
	if *flagDev {
		httpfs = http.Dir(path)
	} else {
		sub, err := fs.Sub(content, path)
		if err != nil {
			log.Fatal(err)
		}
		httpfs = http.FS(sub)
	}
	return http.FileServer(httpfs)
}

var flagDev = flag.Bool("dev", false, "enables reading web files from disk")
var flagAddr = flag.String("addr", ":2103", "listen address")

func main() {
	flag.Parse()

	http.Handle("/", contentFS("html"))
	http.Handle("/static/", http.StripPrefix("/static", contentFS("static")))
	http.Handle("/api/generate", apiJsonHandler(Generate))
	http.Handle("/api/extract", apiJsonHandler(Extract))

	fmt.Printf("listening on http://localhost%s/\n", *flagAddr)
	log.Fatal(http.ListenAndServe(*flagAddr, nil))
}

// Collects errors and returns gzip'd json.
func apiJsonHandler(handler func(r *http.Request) (any, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := handler(r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		gw := gzip.NewWriter(w)
		json.NewEncoder(gw).Encode(data)
		gw.Close()
	})
}

func Extract(r *http.Request) (any, error) {
	f, err := os.Open("../mm3.nsf")
	if err != nil {
		return nil, err
	}
	n, err := nsf.New(f)
	if err != nil {
		return nil, err
	}
	apuState := n.TrackApuState()
	n.Init(5)
	const desired = 1000
	for {
		samples := n.Play(desired)
		if len(samples) < desired {
			break
		}
	}

	var controls []Generated
	for s1, times := range apuState.S1 {
		data := GenerateData{
			Duration: 0,

			P1EnvDuty:           int(s1.EDL & 0b1100_0000 >> 6),
			P1EnvLoop:           s1.EDL&0b10_0000 != 0,
			P1EnvConstantVolume: s1.EDL&0b01_0000 != 0,
			P1EnvVolume:         s1.EnvelopeVolume(),

			P1SweepEnable: s1.Sweep&0b1000_0000 != 0,
			P1SweepPeriod: int(s1.Sweep & 0b111_0000 >> 4),
			P1SweepNegate: s1.Sweep&0b0000_1000 != 0,
			P1SweepShift:  int(s1.Sweep & 0b0111 >> 0),

			P1TimerLength:  0,
			P1TimerCounter: int(s1.Length),
		}
		controls = append(controls, Generated{Data: data, Times: times})
	}
	return controls, nil
}

type Generated struct {
	Data  GenerateData
	Times []nsf.Timerange
}

func Generate(r *http.Request) (any, error) {
	d := json.NewDecoder(r.Body)
	var data GenerateData
	if err := d.Decode(&data); err != nil {
		return nil, err
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
	s1_c4 |= (byte(data.P1EnvLength) & 0b11111) << 3
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
	return vols, nil
}

func bool2byte(b bool) byte {
	if b {
		return 0x1
	}
	return 0x0
}

type GenerateData struct {
	Duration            time.Duration `json:"duration"`
	P1EnvConstantVolume bool          `json:"p1-env-constant-volume"`
	P1EnvDuty           int           `json:"p1-env-duty"`
	P1EnvLength         int           `json:"p1-env-length"`
	P1EnvLoop           bool          `json:"p1-env-loop"`
	P1EnvVolume         int           `json:"p1-env-volume"`
	P1SweepEnable       bool          `json:"p1-sweep-enable"`
	P1SweepNegate       bool          `json:"p1-sweep-negate"`
	P1SweepPeriod       int           `json:"p1-sweep-period"`
	P1SweepShift        int           `json:"p1-sweep-shift"`
	P1TimerCounter      int           `json:"p1-timer-counter"`
	P1TimerLength       int           `json:"p1-timer-length"`
}
