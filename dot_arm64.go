//go:build arm64

package main

func init() {
	int8Dots = append(int8Dots, DotNEON)
}

func DotNEON(a, b []int8) int32
