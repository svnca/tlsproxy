package main

import (
	"fmt"
	"testing"
)

func TestBitsSize(t *testing.T) {
	tests := []struct {
		bps  bitRate
		want string
	}{
		{4 * Mbits, "4Mbits"},
		{2*Mbits + 500*Kbits, "2.5Mbits"},
		{7 * Gbits, "7Gbits"},
		{1000 * Mbits, "1Gbits"},
	}
	for _, tt := range tests {
		tt := tt
		got := fmt.Sprintf("%s", tt.bps)
		if got != tt.want {
			t.Errorf("bitsSizeStr(%d) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}
