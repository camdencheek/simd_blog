package main

type DotF32 func([]float32, []float32) float32
type DotI8 func([]int8, []int8) int32

func DotNaive(a, b []float32) float32 {
	sum := float32(0)
	for i := 0; i < len(a) && i < len(b); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

func DotUnroll4(a, b []float32) float32 {
	if len(a)%4 != 0 {
		panic("slice length must be multiple of 4")
	}

	sum := float32(0)
	for i := 0; i < len(a); i += 4 {
		s0 := a[i] * b[i]
		s1 := a[i+1] * b[i+1]
		s2 := a[i+2] * b[i+2]
		s3 := a[i+3] * b[i+3]
		sum += s0 + s1 + s2 + s3
	}
	return sum
}

func DotUnroll8(a, b []float32) float32 {
	if len(a)%8 != 0 {
		panic("slice length must be multiple of 4")
	}

	sum := float32(0)
	for i := 0; i < len(a); i += 8 {
		s0 := a[i] * b[i]
		s1 := a[i+1] * b[i+1]
		s2 := a[i+2] * b[i+2]
		s3 := a[i+3] * b[i+3]
		s4 := a[i+4] * b[i+4]
		s5 := a[i+5] * b[i+5]
		s6 := a[i+6] * b[i+6]
		s7 := a[i+7] * b[i+7]
		sum += s0 + s1 + s2 + s3 + s4 + s5 + s6 + s7
	}
	return sum
}

func DotBCEOnly(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("slices must have equal lengths")
	}

	if len(a)%4 != 0 {
		panic("slice length must be multiple of 4")
	}

	sum := float32(0)
	for i := 0; i < len(a); i += 1 {
		sum += a[i] * b[i]
	}
	return sum
}

func DotBCE(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("slices must have equal lengths")
	}

	if len(a)%4 != 0 {
		panic("slice length must be multiple of 4")
	}

	sum := float32(0)
	for i := 0; i < len(a); i += 4 {
		aTmp := a[i : i+4 : i+4]
		bTmp := b[i : i+4 : i+4]
		s0 := aTmp[0] * bTmp[0]
		s1 := aTmp[1] * bTmp[1]
		s2 := aTmp[2] * bTmp[2]
		s3 := aTmp[3] * bTmp[3]
		sum += s0 + s1 + s2 + s3
	}
	return sum
}

func DotInt8Naive(a, b []int8) int32 {
	sum := int32(0)
	for i := 0; i < len(a) && i < len(b); i++ {
		sum += int32(a[i]) * int32(b[i])
	}
	return sum
}

func DotInt8Unroll4(a, b []int8) int32 {
	sum := int32(0)
	for i := 0; i < len(a); i += 4 {
		s0 := int32(a[i]) * int32(b[i])
		s1 := int32(a[i+1]) * int32(b[i+1])
		s2 := int32(a[i+2]) * int32(b[i+2])
		s3 := int32(a[i+3]) * int32(b[i+3])
		sum += s0 + s1 + s2 + s3
	}
	return sum
}

func DotInt8Unroll8(a, b []int8) int32 {
	sum := int32(0)
	for i := 0; i < len(a); i += 8 {
		s0 := int32(a[i]) * int32(b[i])
		s1 := int32(a[i+1]) * int32(b[i+1])
		s2 := int32(a[i+2]) * int32(b[i+2])
		s3 := int32(a[i+3]) * int32(b[i+3])
		s4 := int32(a[i+4]) * int32(b[i+4])
		s5 := int32(a[i+5]) * int32(b[i+5])
		s6 := int32(a[i+6]) * int32(b[i+6])
		s7 := int32(a[i+7]) * int32(b[i+7])
		sum += s0 + s1 + s2 + s3 + s4 + s5 + s6 + s7
	}
	return sum
}

func DotInt8BCE(a, b []int8) int32 {
	if len(a) != len(b) {
		panic("slices must have equal lengths")
	}

	if len(a)%4 != 0 {
		panic("slice length must be multiple of 4")
	}

	sum := int32(0)
	for i := 0; i < len(a); i += 4 {
		aTmp := a[i : i+4 : i+4]
		bTmp := b[i : i+4 : i+4]
		s0 := int32(aTmp[0]) * int32(bTmp[0])
		s1 := int32(aTmp[1]) * int32(bTmp[1])
		s2 := int32(aTmp[2]) * int32(bTmp[2])
		s3 := int32(aTmp[3]) * int32(bTmp[3])
		sum += s0 + s1 + s2 + s3
	}
	return sum
}

var f32Dots = []DotF32{
	DotNaive,
	DotUnroll4,
	DotUnroll8,
	DotBCE,
	DotBCEOnly,
}

var int8Dots = []DotI8{
	DotInt8Naive,
	DotInt8Unroll4,
	DotInt8Unroll8,
	DotInt8BCE,
}
