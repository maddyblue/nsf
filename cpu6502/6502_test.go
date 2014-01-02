package cpu6502

import (
	"io/ioutil"
	"testing"
)

type CpuTest struct {
	Name string
	Mem  []byte
	End  Cpu
}

type Ram []byte

func (r Ram) Read(v uint16) byte     { return r[v] }
func (r Ram) Write(v uint16, b byte) { r[v] = b }

var CpuTests = []CpuTest{
	{
		Name: "load, set",
		Mem:  []byte{0xa9, 0x01, 0x8d, 0x00, 0x02, 0xa9, 0x05, 0x8d, 0x01, 0x02, 0xa9, 0x08, 0x8d, 0x02, 0x02},
		End: Cpu{
			A:  0x08,
			S:  0xff,
			PC: 0x0611,
			P:  0x30,
		},
	},
	{
		Name: "load, transfer, increment, add",
		Mem:  []byte{0xa9, 0xc0, 0xaa, 0xe8, 0x69, 0xc4, 0x00},
		End: Cpu{
			A:  0x84,
			X:  0xc1,
			S:  0xff,
			PC: 0x0608,
			P:  0xb1,
		},
	},
	{
		Name: "bne",
		Mem:  []byte{0xa2, 0x08, 0xca, 0x8e, 0x00, 0x02, 0xe0, 0x03, 0xd0, 0xf8, 0x8e, 0x01, 0x02, 0x00},
		End: Cpu{
			X:  0x03,
			S:  0xff,
			PC: 0x060f,
			P:  0x33,
		},
	},
	{
		Name: "relative",
		Mem:  []byte{0xa9, 0x01, 0xc9, 0x02, 0xd0, 0x02, 0x85, 0x22, 0x00},
		End: Cpu{
			A:  0x01,
			S:  0xff,
			PC: 0x060a,
			P:  0xb0,
		},
	},
	{
		Name: "indirect",
		Mem:  []byte{0xa9, 0x01, 0x85, 0xf0, 0xa9, 0xcc, 0x85, 0xf1, 0x6c, 0xf0, 0x00},
		End: Cpu{
			A:  0xcc,
			S:  0xff,
			PC: 0xcc03,
			P:  0xb0,
		},
	},
	{
		Name: "indexed indirect",
		Mem:  []byte{0xa2, 0x01, 0xa9, 0x05, 0x85, 0x01, 0xa9, 0x06, 0x85, 0x02, 0xa0, 0x0a, 0x8c, 0x05, 0x06, 0xa1, 0x00},
		End: Cpu{
			A:  0x0a,
			X:  0x01,
			Y:  0x0a,
			S:  0xff,
			PC: 0x0613,
			P:  0x30,
		},
	},
	{
		Name: "indirect indexed",
		Mem:  []byte{0xa0, 0x01, 0xa9, 0x03, 0x85, 0x01, 0xa9, 0x07, 0x85, 0x02, 0xa2, 0x0a, 0x8e, 0x04, 0x07, 0xb1, 0x01},
		End: Cpu{
			A:  0x0a,
			X:  0x0a,
			Y:  0x01,
			S:  0xff,
			PC: 0x0613,
			P:  0x30,
		},
	},
	{
		Name: "stack",
		Mem:  []byte{0xa2, 0x00, 0xa0, 0x00, 0x8a, 0x99, 0x00, 0x02, 0x48, 0xe8, 0xc8, 0xc0, 0x10, 0xd0, 0xf5, 0x68, 0x99, 0x00, 0x02, 0xc8, 0xc0, 0x20, 0xd0, 0xf7},
		End: Cpu{
			X:  0x10,
			Y:  0x20,
			S:  0xff,
			PC: 0x061a,
			P:  0x33,
		},
	},
	{
		Name: "jsr/rts",
		Mem:  []byte{0x20, 0x09, 0x06, 0x20, 0x0c, 0x06, 0x20, 0x12, 0x06, 0xa2, 0x00, 0x60, 0xe8, 0xe0, 0x05, 0xd0, 0xfb, 0x60, 0x00},
		End: Cpu{
			X:  0x05,
			S:  0xfd,
			PC: 0x0614,
			P:  0x33,
		},
	},
	{
		Name: "others",
		Mem:  []byte{0xa9, 0x30, 0x29, 0x9f, 0x0a, 0xa2, 0x0f, 0x86, 0x00, 0x06, 0x00, 0xa4, 0x00, 0x24, 0x00},
		End: Cpu{
			A:  0x20,
			X:  0x0f,
			Y:  0x1e,
			S:  0xff,
			PC: 0x0611,
			P:  0x32,
		},
	},
	{
		Name: "trb1",
		Mem:  []byte{0xa9, 0xa6, 0x85, 0x00, 0xa9, 0x33, 0x14, 0x00},
		End: Cpu{
			A:  0x33,
			S:  0xff,
			PC: 0x060a,
			P:  0x30,
		},
	},
	{
		Name: "trb2",
		Mem:  []byte{0xa9, 0xa6, 0x85, 0x00, 0xa9, 0x41, 0x14, 0x00},
		End: Cpu{
			A:  0x41,
			S:  0xff,
			PC: 0x060a,
			P:  0x32,
		},
	},
	{
		Name: "tsb1",
		Mem:  []byte{0xa9, 0xa6, 0x85, 0x00, 0xa9, 0x33, 0x04, 0x00},
		End: Cpu{
			A:  0x33,
			S:  0xff,
			PC: 0x060a,
			P:  0x30,
		},
	},
	{
		Name: "tsb2",
		Mem:  []byte{0xa9, 0xa6, 0x85, 0x00, 0xa9, 0x41, 0x04, 0x00},
		End: Cpu{
			A:  0x41,
			S:  0xff,
			PC: 0x060a,
			P:  0x32,
		},
	},
}

func Test6502(t *testing.T) {
	for _, test := range CpuTests {
		r := make(Ram, 0xffff+1)
		c := New(r)
		copy(r[c.PC:], test.Mem)
		c.Run()
		if c.A != test.End.A ||
			c.X != test.End.X ||
			c.Y != test.End.Y ||
			c.S != test.End.S ||
			c.PC != test.End.PC ||
			c.P != test.End.P {
			t.Fatalf("bad cpu state %s, got:\n%sexpected:\n%s", test.Name, c, &test.End)
		}
	}
}

// Download from https://github.com/Klaus2m5/6502_65C02_functional_tests/blob/master/bin_files/6502_functional_test.bin
// GPL, so not included here.
func TestFunctional(t *testing.T) {
	b, err := ioutil.ReadFile("6502_functional_test.bin")
	if err != nil {
		t.Fatal(err)
	}
	r := make(Ram, 0xffff+1)
	copy(r[:], b)
	c := New(r)
	c.PC = 0x0400
	for !c.Halt {
		pc := c.PC
		c.Step()
		if c.PC == pc {
			t.Fatal()
		}
	}
}
