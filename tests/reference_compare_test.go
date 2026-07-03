package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestCompareReferenceCodewords(t *testing.T) {
	refData, err := os.ReadFile("./test_1600.wav")
	if err != nil {
		t.Skip("./test_1600.wav not available")
	}
	frames, err := flex.DemodulateRawFrames(refData)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("reference: %d raw frames", len(frames))

	ourWav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}}, flex.Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	ourFrames, err := flex.DemodulateRawFrames(ourWav)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ours: %d raw frames", len(ourFrames))

	for _, rf := range frames {
		info1, _ := flex.FLEXBCHDecode32(rf.Words[1])
		capcode := int64(info1&0x1FFFFF) - 0x8000
		if capcode != 1913 {
			continue
		}
		t.Logf("=== reference frame cycle=%d frame=%d capcode=%d ===", rf.Cycle, rf.Frame, capcode)
		for i := 0; i < 8; i++ {
			logWire(t, "ref", i, rf.Words[i])
		}
		if len(ourFrames) > 0 {
			of := ourFrames[0]
			t.Log("=== our frame 0 ===")
			for i := 0; i < 8; i++ {
				logWire(t, "our", i, of.Words[i])
				if rf.Words[i] != of.Words[i] {
					t.Logf("  MISMATCH cw[%d] ref=0x%08X our=0x%08X", i, rf.Words[i], of.Words[i])
				}
			}
		}
		return
	}
	t.Fatal("no reference frame with capcode 1913")
}

func logWire(t *testing.T, tag string, idx int, wire uint32) {
	logical, errs := flex.FLEXBCHDecode32(wire)
	t.Logf("%s cw[%d] wire=0x%08X decode=0x%05X errs=%d xsum=%v",
		tag, idx, wire, logical, errs, flex.ExportFLEXChecksum(logical))
}
