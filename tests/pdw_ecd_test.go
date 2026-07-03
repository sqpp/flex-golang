package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestPDWDecodeOurFrame(t *testing.T) {
	ourWav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}}, flex.Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	frames, _ := flex.DemodulateRawFrames(ourWav)
	if len(frames) == 0 {
		t.Fatal("no frames")
	}

	refData, _ := os.ReadFile("./test_1600.wav")
	refFrames, _ := flex.DemodulateRawFrames(refData)

	var refFrame *flex.RawPhaseFrame
	for i := range refFrames {
		info, _ := flex.FLEXBCHDecode32(refFrames[i].Words[1])
		if int64(info&0x1FFFFF)-0x8000 == 1913 {
			refFrame = &refFrames[i]
			break
		}
	}

	of := frames[0]
	for i := 0; i < 10; i++ {
		goLog, goErrs := flex.FLEXBCHDecode32(of.Words[i])
		pdwLog, pdwErr := flex.PDWDecodeWire(of.Words[i])
		t.Logf("our cw[%d] wire=0x%08X go=0x%05X errs=%d xsum=%v pdw=0x%06X err=%d xsum=%v match=%v",
			i, of.Words[i], goLog, goErrs, flex.ExportFLEXChecksum(goLog),
			pdwLog&0x1FFFFF, pdwErr, flex.ExportPDWXsumchk(pdwLog),
			(int64(goLog)&0x1FFFFF) == (pdwLog&0x1FFFFF))

		if refFrame != nil {
			refLog, refErr := flex.PDWDecodeWire(refFrame.Words[i])
			t.Logf("  ref pdw=0x%06X err=%d xsum=%v", refLog&0x1FFFFF, refErr, flex.ExportPDWXsumchk(refLog))
		}
	}

	biw, _ := flex.PDWDecodeWire(of.Words[0])
	if !flex.ExportPDWXsumchk(biw) {
		t.Error("BIW fails PDW xsumchk")
	}
	vec, _ := flex.PDWDecodeWire(of.Words[2])
	if !flex.ExportPDWXsumchk(vec) {
		t.Error("vector fails PDW xsumchk")
	}
}
