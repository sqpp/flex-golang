package flex_test

import (
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestPDWSyncMapping(t *testing.T) {
	cases := []struct {
		mode     string
		syncCode uint16
		symRate  int
		bitRate  int
		levels   int
	}{
		{flex.Mode1600_2, 0x870C, 1600, 1600, 2},
		{flex.Mode3200_2, 0x7B18, 3200, 3200, 2},
		{flex.Mode3200_4, 0xB068, 1600, 3200, 4},
		{flex.Mode1600_4, 0xB068, 1600, 3200, 4},
		{flex.Mode6400_4, 0xDEA0, 3200, 6400, 4},
	}
	for _, tc := range cases {
		m, err := flex.LookupEncodeMode(tc.mode)
		if err != nil {
			t.Fatalf("%s: %v", tc.mode, err)
		}
		if m.SyncCode != tc.syncCode || m.SymRate != tc.symRate || m.Levels != tc.levels {
			t.Fatalf("%s: mode=%+v want sync=0x%04X sym=%d lev=%d",
				tc.mode, m, tc.syncCode, tc.symRate, tc.levels)
		}
		if m.BitRate != tc.bitRate {
			t.Fatalf("%s: bitRate=%d want %d", tc.mode, m.BitRate, tc.bitRate)
		}
	}
}
