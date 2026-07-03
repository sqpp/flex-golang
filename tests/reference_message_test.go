package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestEncodeReferenceMessage(t *testing.T) {
	refData, err := os.ReadFile("./test_1600.wav")
	if err != nil {
		t.Skip("./test_1600.wav not available")
	}

	var refFrame *flex.RawPhaseFrame
	refFrames, err := flex.DemodulateRawFrames(refData)
	if err != nil {
		t.Fatal(err)
	}
	for i := range refFrames {
		info, _ := flex.FLEXBCHDecode32(refFrames[i].Words[1])
		if int64(info&0x1FFFFF)-0x8000 == 1913 {
			refFrame = &refFrames[i]
			t.Logf("reference frame cycle=%d frame=%d", refFrames[i].Cycle, refFrames[i].Frame)
			break
		}
	}
	if refFrame == nil {
		t.Fatal("no reference frame for capcode 1913")
	}

	ourWav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{{
		Capcode: 1913,
		Type:    "alpha",
		Text:    flex.ReferenceMessage1913,
	}}, flex.Mode1600_2, refFrame.Cycle, refFrame.Frame)
	if err != nil {
		t.Fatal(err)
	}
	ourFrames, err := flex.DemodulateRawFrames(ourWav)
	if err != nil {
		t.Fatal(err)
	}
	if len(ourFrames) == 0 {
		t.Fatal("no encoded frames")
	}
	of := ourFrames[0]

	msgs, err := flex.DecodeFromAudio(ourWav)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d: %+v", len(msgs), msgs)
	}
	t.Logf("our decode text=%q", msgs[0].Text)
	if msgs[0].Text != flex.ReferenceMessage1913 {
		t.Errorf("text mismatch:\n got: %q\nwant: %q", msgs[0].Text, flex.ReferenceMessage1913)
	}

	matches := 0
	for i := 0; i < 12; i++ {
		refWire := refFrame.Words[i]
		ourWire := of.Words[i]
		refLog, _ := flex.FLEXBCHDecode32(refWire)
		ourLog, _ := flex.FLEXBCHDecode32(ourWire)
		match := refWire == ourWire
		if match {
			matches++
		}
		t.Logf("cw[%2d] ref=0x%08X our=0x%08X wire_match=%v ref_log=0x%05X our_log=0x%05X",
			i, refWire, ourWire, match, refLog, ourLog)
	}
	t.Logf("wire matches in first 12 codewords: %d/12", matches)
}
