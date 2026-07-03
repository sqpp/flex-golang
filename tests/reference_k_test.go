package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestReferenceHeaderK(t *testing.T) {
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
	if refFrame == nil {
		t.Fatal("no ref frame")
	}

	refHeader, _ := flex.FLEXBCHDecode32(refFrame.Words[3])
	targetK := refHeader & 0x7FF
	t.Logf("reference header=0x%05X K=0x%03X", refHeader, targetK)

	content := flex.ExportEncodeAlphaPayload(flex.ReferenceMessage1913, 84, true)
	if len(content) == 0 {
		t.Fatal("no content")
	}

	type try struct {
		name string
		fn   func() uint32
	}
	tries := []try{
		{"after_sig", func() uint32 {
			c := append([]uint32(nil), content...)
			c[0] |= flex.ExportAlphaSignature(c)
			return flex.ExportAlphaHeaderChecksum(0x1800, c)
		}},
		{"before_sig", func() uint32 {
			return flex.ExportAlphaHeaderChecksum(0x1800, content)
		}},
		{"header_zero", func() uint32 {
			c := append([]uint32(nil), content...)
			c[0] |= flex.ExportAlphaSignature(c)
			return flex.ExportAlphaHeaderChecksum(0, c)
		}},
		{"with_frag_in_sum", func() uint32 {
			c := append([]uint32(nil), content...)
			c[0] |= flex.ExportAlphaSignature(c)
			kSum := uint32(0)
			hw := uint32(0x1800)
			kSum += (hw & 0xFF) + ((hw >> 8) & 0xFF) + ((hw >> 16) & 0x1F)
			for _, w := range c {
				kSum += w & 0x1FFFFF
			}
			return (^kSum) & 0x3FF
		}},
	}

	for _, tr := range tries {
		k := tr.fn()
		t.Logf("%s -> K=0x%03X match=%v", tr.name, k, k == targetK)
	}
}
