package flex_test

import (
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestSyncInBitstream(t *testing.T) {
	header, err := flex.ExportBuildBitstreamHeader(flex.EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "X",
	}, flex.Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	var syncReg uint64
	syncAt := -1
	for i, bit := range header {
		b := uint64(0)
		if bit == 0 {
			b = 1
		}
		syncReg = (syncReg << 1) | b
		if i >= 63 {
			if _, ok := flex.ExportCheckSync1(syncReg); ok {
				syncAt = i
				break
			}
		}
	}
	if syncAt < 0 {
		t.Fatal("sync not found")
	}
	t.Logf("sync at bit %d (header symbols %d)", syncAt, len(header))
}

func TestPushPerfectSquares(t *testing.T) {
	wav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}}, flex.Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := flex.DecodeFromAudio(wav)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Fatal("no sync on encoded WAV")
	}
	t.Logf("decoded: %+v", msgs[0])
}
