package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateTestFloats(dims, count int) []float32 {
	data := make([]float32, dims*count)
	for i := 0; i < len(data); i++ {
		data[i] = rand.Float32()
	}
	return data
}

func generateTestInts(dims, count int) []int8 {
	data := make([]int8, dims*count)
	for i := 0; i < len(data); i++ {
		data[i] = int8(rand.Int())
	}
	return data
}

const dims = 1536

var (
	testDataF32  = generateTestFloats(dims, 512*1024) // 3GiB
	testProbeF32 = generateTestFloats(dims, 1)
	testDataI8   = generateTestInts(dims, 512*1024) // 3GiB
	testProbeI8  = generateTestInts(dims, 1)

	blackholeF32 float32
	blackholeI32 int32
)

func TestDot(t *testing.T) {
	naive := DotNaive(testProbeF32, testDataF32[:dims])

	unroll4 := DotUnroll4(testProbeF32, testDataF32[:dims])
	require.InEpsilon(t, naive, unroll4, 0.01)

	unroll8 := DotUnroll8(testProbeF32, testDataF32[:dims])
	require.InEpsilon(t, naive, unroll8, 0.01)

	bce := DotBCE(testProbeF32, testDataF32[:dims])
	require.InEpsilon(t, naive, bce, 0.01)

	naiveInt8 := DotNaiveInt8(testProbeI8, testDataI8[:dims])

	optimizedInt8 := DotInt8(testProbeI8, testDataI8[:dims])
	require.Equal(t, naiveInt8, optimizedInt8)

	simdInt8 := DotSIMD(testProbeI8, testDataI8[:dims])
	require.Equal(t, naiveInt8, simdInt8)
}

func BenchmarkDot(b *testing.B) {
	b.Run("naive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataF32); i += dims {
				blackholeF32 = DotNaive(testProbeF32, testDataF32[i:i+dims])
			}
		}
	})

	b.Run("unroll4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataF32); i += dims {
				blackholeF32 = DotUnroll4(testProbeF32, testDataF32[i:i+dims])
			}
		}
	})

	b.Run("unroll8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataF32); i += dims {
				blackholeF32 = DotUnroll8(testProbeF32, testDataF32[i:i+dims])
			}
		}
	})

	b.Run("bce", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataF32); i += dims {
				blackholeF32 = DotBCE(testProbeF32, testDataF32[i:i+dims])
			}
		}
	})

	b.Run("naive int8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataI8); i += dims {
				blackholeI32 = DotNaiveInt8(testProbeI8, testDataI8[i:i+dims])
			}
		}
	})

	b.Run("int8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataI8); i += dims {
				blackholeI32 = DotInt8(testProbeI8, testDataI8[i:i+dims])
			}
		}
	})

	b.Run("simd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for i := 0; i < len(testDataI8); i += dims {
				blackholeI32 = DotSIMD(testProbeI8, testDataI8[i:i+dims])
			}
		}
	})
}
