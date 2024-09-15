package nsf

import (
	"os"
	"testing"
	"time"

	"github.com/ebitengine/oto/v3"
)

func TestNsf(t *testing.T) {
	testNsf(t, "mm3.nsf", 1)
}

func TestNsfe(t *testing.T) {
	testNsf(t, "mm3.nsfe", 11)
}

func testNsf(t *testing.T, name string, idx int) {
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	n, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	if n.LoadAddr != 0x8000 || n.InitAddr != 0x8003 || n.PlayAddr != 0x8000 {
		t.Fatal("bad addresses")
	}
	n.Init(idx)

	op := &oto.NewContextOptions{}
	op.SampleRate = int(n.SampleRate)
	op.ChannelCount = 1
	op.Format = oto.FormatFloat32LE

	if otoCtx == nil {
		ctx, readyChan, err := oto.NewContext(op)
		if err != nil {
			t.Fatal("oto.NewContext failed: " + err.Error())
		}
		<-readyChan
		otoCtx = ctx
	}
	player := otoCtx.NewPlayer(n)
	player.Play()
	time.Sleep(time.Second * 10)
	if !player.IsPlaying() {
		t.Fatal("not playing")
	}
	if player.Err() != nil {
		t.Fatal("player err", player.Err())
	}
	player.Close()
}

var otoCtx *oto.Context
