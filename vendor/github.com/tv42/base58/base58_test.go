package base58

import (
	"fmt"
	"math/big"
	"testing"
)

type testpair struct {
	decoded int64
	encoded string
}

var pairs = []testpair{
	{10002343, "Tgmc"},
	{1000, "if"},
	{0, ""},
}

func ExampleEncodeBig() {
	buf := EncodeBig(nil, big.NewInt(123456))
	fmt.Printf("%s\n", buf)
	// Output:
	// CGy
}

func ExampleDecodeToBig() {
	n, err := DecodeToBig([]byte("CGy"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%d\n", n)
	// Output:
	// 123456
}

func TestEncode(t *testing.T) {
	for _, p := range pairs {
		var buf []byte = ([]byte)("noise")
		buf = EncodeBig(buf, big.NewInt(p.decoded))
		if string(buf) != "noise"+p.encoded {
			t.Errorf("unexpected result: %q != %q", string(buf), p.encoded)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, data := range pairs {
		var buf []byte = []byte(data.encoded)
		n, err := DecodeToBig(buf)
		if err != nil {
			t.Errorf("decoding %q failed: %v", data.encoded, err)
		}
		if n.Int64() != data.decoded {
			t.Errorf("unexpected result: %v != %v", n, data.decoded)
		}
	}
}

func TestDecodeCorrupt(t *testing.T) {
	type corrupt struct {
		input  string
		offset int
	}
	examples := []corrupt{
		{"!!!!", 0},
		{"x===", 1},
		{"x0", 1},
		{"xl", 1},
		{"xI", 1},
		{"xO", 1},
	}

	for _, e := range examples {
		_, err := DecodeToBig([]byte(e.input))
		switch err := err.(type) {
		case CorruptInputError:
			if int(err) != e.offset {
				t.Errorf("Corruption in %q at offset %v, want %v", e.input, int(err), e.offset)
			}
		default:
			t.Error("Decoder failed to detect corruption in", e)
		}
	}
}
