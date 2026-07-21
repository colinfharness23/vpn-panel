package model

import "testing"

func TestApplyTrafficMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		permille int
		want     int64
	}{
		{name: "legacy zero is one x", bytes: 1024, permille: 0, want: 1024},
		{name: "half rate", bytes: 1024, permille: 500, want: 512},
		{name: "two x", bytes: 1024, permille: 2000, want: 2048},
		{name: "fraction rounds up", bytes: 1, permille: 1500, want: 2},
		{name: "zero bytes", bytes: 0, permille: 2000, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyTrafficMultiplier(tt.bytes, tt.permille); got != tt.want {
				t.Fatalf("ApplyTrafficMultiplier(%d, %d) = %d, want %d", tt.bytes, tt.permille, got, tt.want)
			}
		})
	}
}
