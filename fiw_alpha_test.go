package flex

import (
	"os"
	"testing"
)

func TestFIWAndAlphaLayout(t *testing.T) {
	ref, _ := os.ReadFile("tests/test_1600.wav")
	frames, _ := DemodulateRawFrames(ref)
	for _, rf := range frames {
		info, _ := FLEXBCHDecode32(rf.Words[1])
		if int64(info&0x1FFFFF)-0x8000 != 1913 {
			continue
		}
		vec, _ := FLEXBCHDecode32(rf.Words[2])
		w1 := (vec >> 7) & 0x7F
		w2 := ((vec >> 14) & 0x7F) + w1 - 1
		t.Logf("ref frame cycle=%d frame=%d", rf.Cycle, rf.Frame)
		t.Logf("ref vector logical=0x%05X start=%d end=%d type=%d", vec, w1, w2, (vec>>4)&7)
		for i := int(w1); i <= int(w2) && i < 12; i++ {
			log, _ := FLEXBCHDecode32(rf.Words[i])
			t.Logf("  ref cw[%d] wire=0x%08X logical=0x%05X", i, rf.Words[i], log)
		}
		break
	}

	ourBits, _ := BuildBitstream(EncodeMessage{Capcode: 1913, Type: "alpha", Text: "HELLO WORLD"}, Mode1600_2, 0, 0)
	ourWav := modulateBits(ourBits, 1600)
	ourFrames, _ := DemodulateRawFrames(ourWav)
	if len(ourFrames) == 0 {
		t.Fatal("no our frames")
	}
	of := ourFrames[0]
	vec, _ := FLEXBCHDecode32(of.Words[2])
	w1 := (vec >> 7) & 0x7F
	w2 := ((vec >> 14) & 0x7F) + w1 - 1
	t.Logf("our frame cycle=%d frame=%d", of.Cycle, of.Frame)
	t.Logf("our vector logical=0x%05X start=%d end=%d type=%d", vec, w1, w2, (vec>>4)&7)
	for i := 0; i < 10; i++ {
		log, _ := FLEXBCHDecode32(of.Words[i])
		t.Logf("  our cw[%d] wire=0x%08X logical=0x%05X", i, of.Words[i], log)
	}

	msgs, _ := DecodeFromAudio(ourWav)
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
		{"refLayout_LSB", flexEncodeWord(buildFIWData(0, 0)), false},
		{"refLayout_MSB", flexEncodeWord(buildFIWData(0, 0)), true},
		{"direct_LSB", encodeFIWDirect(buildFIWData(0, 0)), false},
		{"direct_MSB", encodeFIWDirect(buildFIWData(0, 0)), true},
	} {
		bits, _ := bitstreamFromCodewordsWithFIW(tc.fiw, tc.msb)
		wav := modulateBits(bits, 1600)
		frames, _ := DemodulateRawFrames(wav)
		cy, fr := -1, -1
		if len(frames) > 0 {
			cy, fr = frames[0].Cycle, frames[0].Frame
		}
		t.Logf("%s fiw=0x%08X msb=%v -> demod cycle=%d frame=%d frames=%d",
			tc.name, tc.fiw, tc.msb, cy, fr, len(frames))
	}
}

func encodeFIWDirect(logical21 uint32) uint32 {
	poc := BCHEncode31_21(logical21 & 0x1FFFFF)
	cw := (poc >> 10) | ((poc & 0x3FF) << 21)
	if popCount32(cw)&1 != 0 {
		cw |= 1 << 31
	}
	return cw
}

func bitstreamFromCodewordsWithFIW(fiw uint32, fiwMSB bool) ([]byte, error) {
	codewords, mode, err := assembleCodewords(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}, Mode1600_2, 0, 0)
	if err != nil {
		return nil, err
	}
	var bits []byte
	for i := 0; i < 960; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}
	appendBitsMSBInv(&bits, buildSync1(mode.syncCode), 64)
	for i := 0; i < 16; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}
	if fiwMSB {
		appendBitsMSB(&bits, fiw, 32)
	} else {
		appendBitsLSB(&bits, fiw, 32)
	}
	for i := 0; i < 4; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}
	const cPat = uint16(0xED84)
	for i := 15; i >= 0; i-- {
		bits = append(bits, byte((cPat>>uint(i))&1))
	}
	for i := 0; i < 4; i++ {
		if i&1 == 0 {
			bits = append(bits, 1)
		} else {
			bits = append(bits, 0)
		}
	}
	const cInvPat = uint16(0x127B)
	for i := 15; i >= 0; i-- {
		bits = append(bits, byte((cInvPat>>uint(i))&1))
	}
	appendInterleavedData(&bits, codewords)
	for i := 0; i < 64; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}
	return bits, nil
}

func appendBitsMSB(bits *[]byte, value uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		*bits = append(*bits, byte((value>>uint(i))&1))
	}
}
