//go:build amd64

package main

func init() {
	int8Dots = append(int8Dots, DotAVX2, DotVNNI)
}

func DotAVX2(a, b []int8) int32
func DotVNNI(a, b []int8) int32
