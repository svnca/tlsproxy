package main

import (
	"fmt"
)

type bitRate int64

const (
	Bits  bitRate = 1
	Kbits         = 1000 * Bits
	Mbits         = 1000 * Kbits
	Gbits         = 1000 * Mbits
	Tbits         = 1000 * Gbits

	ToBits = 8
)

func (b bitRate) String() string {
	return bitrateStr(float64(b))
}

var (
	bitspsAbbrs = []string{"bits", "Kbits", "Mbits", "Gbits", "Tbits"}
)

// bitrateStr returns a human-readable size in bits, kbits,
// megabits, gigabits (eg. "44Kbits", "17Mbits").
func bitrateStr(size float64) string {
	return customSize("%.4g%s", size, 1000.0, bitspsAbbrs)
}

// customSize returns a human-readable approximation of a size
// using custom format.
func customSize(format string, size float64, base float64, _map []string) string {
	size, unit := getSizeAndUnit(size, base, _map)
	return fmt.Sprintf(format, size, unit)
}

func getSizeAndUnit(size float64, base float64, _map []string) (float64, string) {
	i := 0
	unitsLimit := len(_map) - 1
	for size >= base && i < unitsLimit {
		size = size / base
		i++
	}
	return size, _map[i]
}
