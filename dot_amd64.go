//go:build amd64

package main

import (
// "github.com/klauspost/cpuid/v2"
)

func init() {
	// hasAVX2 := cpuid.CPU.Has(cpuid.AVX2)
	// hasVNNI := cpuid.CPU.Supports(
	// 	cpuid.AVX512F,    // required by VPXORQ, VPSUBD, VPBROADCASTD
	// 	cpuid.AVX512BW,   // required by VMOVDQU8, VADDB, VPSRLDQ
	// 	cpuid.AVX512VNNI, // required by VPDPBUSD
	// )

	// if simdEnabled && hasVNNI {
	// 	dotArch = dotVNNI
	// } else if simdEnabled && hasAVX2 {
	// 	dotArch = dotAVX2
	// }

	DotSIMD = DotVNNI
}

func DotAVX2(a, b []int8) int32

func DotVNNI(a, b []int8) int32
