package base58_test

import (
	"math/big"
	"testing"

	"github.com/cmars/basen"
	"github.com/tv42/base58"
)

// These benchmarks mirror the ones in encoding/base64, and results
// should be comparable to those.

func BenchmarkBase58EncodeToString(b *testing.B) {
	data := make([]byte, 8192)
	data[0] = 0xff // without this, it's just an inefficient zero
	b.SetBytes(int64(len(data)))
	var num big.Int
	num.SetBytes(data)
	buf := make([]byte, 12000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base58.EncodeBig(buf[:0], &num)
	}
}

func BenchmarkBase58DecodeString(b *testing.B) {
	data := make([]byte, 8192)
	data[0] = 0xff // without this, it's just an inefficient zero
	data = []byte(basen.Base58.EncodeToString(data))
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base58.DecodeToBig(data)
	}
}

// These benchmarks are more like the uses that this library was meant
// for: smallish identifiers.

func BenchmarkBase58EncodeToStringSmall(b *testing.B) {
	data := make([]byte, 8)
	data[0] = 0xff // without this, it's just an inefficient zero
	b.SetBytes(int64(len(data)))
	var num big.Int
	num.SetBytes(data)
	buf := make([]byte, 12000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base58.EncodeBig(buf[:0], &num)
	}
}

func BenchmarkBase58DecodeStringSmall(b *testing.B) {
	data := make([]byte, 8)
	data[0] = 0xff // without this, it's just an inefficient zero
	data = []byte(basen.Base58.EncodeToString(data))
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base58.DecodeToBig(data)
	}
}
