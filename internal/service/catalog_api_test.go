package service

import "testing"

func TestInsumoSearchLimitClamp(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 20},
		{-1, 20},
		{10, 10},
		{50, 50},
		{100, 50},
	}
	for _, tc := range cases {
		got := clampSupplySearchLimit(tc.in)
		if got != tc.want {
			t.Fatalf("clampSupplySearchLimit(%d)=%d want %d", tc.in, got, tc.want)
		}
	}
}
