package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestFIWAndAlphaLayout(t *testing.T) {
	ref, _ := os.ReadFile("./test_1600.wav")
	frames, _ := flex.DemodulateRawFrames(ref)
	for _, rf := range frames {
		info, _ := flex.FLEXBCHDecode32(rf.Words[1])
		if int64(info&0x1FFFFF)-0x8000 != 1913 {
			continue
		}
		vec, _ := flex.FLEXBCHDecode32(rf.Words[2])
		w1 := (vec >> 7) & 0x7F
		w2 := ((vec >> 14) & 0x7F) + w1 - 1
		t.Logf("ref frame cycle=%d frame=%d", rf.Cycle, rf.Frame)
		t.Logf("ref vector logical=0x%05X start=%d end=%d type=%d", vec, w1, w2, (vec>>4)&7)
		for i := int(w1); i <= int(w2) && i < 12; i++ {
			log, _ := flex.FLEXBCHDecode32(rf.Words[i])
			t.Logf("  ref cw[%d] wire=0x%08X logical=0x%05X", i, rf.Words[i], log)
		}
		break
	}

	ourWav, _, _, _ := flex.EncodeToWAVBytes([]flex.EncodeMessage{{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}}, flex.Mode1600_2, 0, 0)
	ourFrames, _ := flex.DemodulateRawFrames(ourWav)
	if len(ourFrames) == 0 {
		t.Fatal("no our frames")
	}
	of := ourFrames[0]
	vec, _ := flex.FLEXBCHDecode32(of.Words[2])
	w1 := (vec >> 7) & 0x7F
	w2 := ((vec >> 14) & 0x7F) + w1 - 1
	t.Logf("our frame cycle=%d frame=%d", of.Cycle, of.Frame)
	t.Logf("our vector logical=0x%05X start=%d end=%d type=%d", vec, w1, w2, (vec>>4)&7)
	for i := 0; i < 10; i++ {
		log, _ := flex.FLEXBCHDecode32(of.Words[i])
		t.Logf("  our cw[%d] wire=0x%08X logical=0x%05X", i, of.Words[i], log)
	}

	msgs, _ := flex.DecodeFromAudio(ourWav)
	for _, m := range msgs {
		t.Logf("our decode: cap=%d text=%q", m.Capcode, m.Text)
	}
}

func TestFIWBitOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		fiw  uint32
		msb  bool
	}{
		{"refLayout_LSB", flex.ExportFlexEncodeWord(flex.ExportBuildFIWData(0, 0)), false},
		{"refLayout_MSB", flex.ExportFlexEncodeWord(flex.ExportBuildFIWData(0, 0)), true},
		{"direct_LSB", flex.ExportEncodeFIWDirect(flex.ExportBuildFIWData(0, 0)), false},
		{"direct_MSB", flex.ExportEncodeFIWDirect(flex.ExportBuildFIWData(0, 0)), true},
	} {
		wav, err := flex.ExportBitstreamWithFIW(tc.fiw, tc.msb)
		if err != nil {
			t.Fatal(err)
		}
		frames, _ := flex.DemodulateRawFrames(wav)
		cy, fr := -1, -1
		if len(frames) > 0 {
			cy, fr = frames[0].Cycle, frames[0].Frame
		}
		t.Logf("%s fiw=0x%08X msb=%v -> demod cycle=%d frame=%d frames=%d",
			tc.name, tc.fiw, tc.msb, cy, fr, len(frames))
	}
}
