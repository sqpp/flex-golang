package flex

import (
	"os"
	"testing"
)

func TestPDWDecodeOurFrame(t *testing.T) {
	ourBits, _ := BuildBitstream(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}, Mode1600_2, 0, 0)
	ourWav := modulateBits(ourBits, 1600)
	frames, _ := DemodulateRawFrames(ourWav)
	if len(frames) == 0 {
		t.Fatal("no frames")
	}

	refData, _ := os.ReadFile("tests/test_1600.wav")
	refFrames, _ := DemodulateRawFrames(refData)

	var refFrame *RawPhaseFrame
	for i := range refFrames {
		info, _ := FLEXBCHDecode32(refFrames[i].Words[1])
		if int64(info&0x1FFFFF)-0x8000 == 1913 {
			refFrame = &refFrames[i]
			break
		}
	}

	of := frames[0]
	for i := 0; i < 10; i++ {
		goLog, goErrs := FLEXBCHDecode32(of.Words[i])
		pdwLog, pdwErr := PDWDecodeWire(of.Words[i])
		t.Logf("our cw[%d] wire=0x%08X go=0x%05X errs=%d xsum=%v pdw=0x%06X err=%d xsum=%v match=%v",
			i, of.Words[i], goLog, goErrs, FLEXChecksum(goLog),
			pdwLog&0x1FFFFF, pdwErr, pdwXsumchk(pdwLog),
			(int64(goLog)&0x1FFFFF) == (pdwLog&0x1FFFFF))

		if refFrame != nil {
			refLog, refErr := PDWDecodeWire(refFrame.Words[i])
			t.Logf("  ref pdw=0x%06X err=%d xsum=%v", refLog&0x1FFFFF, refErr, pdwXsumchk(refLog))
		}
	}

	biw, _ := PDWDecodeWire(of.Words[0])
	if !pdwXsumchk(biw) {
		t.Error("BIW fails PDW xsumchk")
	}
	vec, _ := PDWDecodeWire(of.Words[2])
	if !pdwXsumchk(vec) {
		t.Error("vector fails PDW xsumchk")
	}
}
