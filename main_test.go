package main

import (
	"math/rand"
	"reflect"
	"runtime"
	"strings"
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
)

func TestDot(t *testing.T) {
	t.Run("float32", func(t *testing.T) {
		expected := DotNaive(testProbeF32, testDataF32[:dims])
		for _, f := range f32Dots {
			t.Run(funcName(f), func(t *testing.T) {
				got := f(testProbeF32, testDataF32[:dims])
				require.InEpsilon(t, expected, got, 0.01)
			})
		}
	})
	t.Run("int8", func(t *testing.T) {
		expected := DotInt8Naive(testProbeI8, testDataI8[:dims])
		for _, f := range int8Dots {
			t.Run(funcName(f), func(t *testing.T) {
				got := f(testProbeI8, testDataI8[:dims])
				require.Equal(t, expected, got)
			})
		}
	})
}

var (
	// Global destination vars to keep benchmarks
	// from getting optimized away.
	blackholeF32 float32
	blackholeI32 int32
)

func benchmarkF32(b *testing.B, f DotF32) {
	searchedVectors := 0
	for i := 0; i < b.N; i++ {
		for i := 0; i < len(testDataF32); i += dims {
			blackholeF32 = f(testProbeF32, testDataF32[i:i+dims])
		}
		searchedVectors += len(testDataF32) / dims
	}
	b.ReportMetric(float64(searchedVectors)/float64(b.Elapsed().Seconds()), "vecs/s")
}

func benchmarkI8(b *testing.B, f DotI8) {
	searchedVectors := 0
	for i := 0; i < b.N; i++ {
		for i := 0; i < len(testDataF32); i += dims {
			blackholeI32 = f(testProbeI8, testDataI8[i:i+dims])
		}
		searchedVectors += len(testDataF32) / dims
	}
	b.ReportMetric(float64(searchedVectors)/float64(b.Elapsed().Seconds()), "vecs/s")
}

func BenchmarkDot(b *testing.B) {
	for _, f := range f32Dots {
		b.Run(funcName(f), func(b *testing.B) { benchmarkF32(b, f) })
	}
	for _, f := range int8Dots {
		b.Run(funcName(f), func(b *testing.B) { benchmarkI8(b, f) })
	}
}

func funcName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}
