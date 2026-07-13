package handler

import "testing"

// Money rule: rupees -> paisa must round, never truncate (a bare int64 cast
// of 4.35*100 = 434.999... would store 434).
func TestAmountToInt64Rounds(t *testing.T) {
	tests := []struct {
		rupees float64
		paisa  int64
	}{
		{0.01, 1},
		{1, 100},
		{4.35, 435},   // classic float trap: 4.35*100 = 434.999...
		{10.55, 1055},
		{29.99, 2999},
		{1234.56, 123456},
		{0.005, 1}, // half-paisa rounds up
	}
	for _, tc := range tests {
		if got := amountToInt64(tc.rupees); got != tc.paisa {
			t.Errorf("amountToInt64(%v) = %d, want %d", tc.rupees, got, tc.paisa)
		}
	}
}

func TestAmountRoundTrip(t *testing.T) {
	// every representable paisa value must survive paisa -> rupees -> paisa
	for _, paisa := range []int64{1, 99, 100, 435, 1055, 99999, 12345678} {
		rupees := amountToFloat64(paisa)
		if back := amountToInt64(rupees); back != paisa {
			t.Errorf("round trip %d -> %v -> %d", paisa, rupees, back)
		}
	}
}
