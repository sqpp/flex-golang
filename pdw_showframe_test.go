package flex

import (
	"testing"
)

func TestPDWShowframeSimulation(t *testing.T) {
	ourBits, _ := BuildBitstream(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}, Mode1600_2, 3, 111)
	wav := modulateBits(ourBits, 1600)
	frames, _ := DemodulateRawFrames(wav)
	if len(frames) == 0 {
		t.Fatal("no frames")
	}

	frame := make([]int64, PhaseWords)
	for i := 0; i < PhaseWords; i++ {
		log, _ := PDWDecodeWire(frames[0].Words[i])
		frame[i] = log
	}

	if !pdwXsumchk(frame[0]) {
		t.Fatal("BIW xsum fail")
	}

	asa := int((frame[0]>>8)&0x03) + 1
	vsa := int((frame[0] >> 10) & 0x3F)
	if vsa <= asa {
		t.Fatalf("empty frame asa=%d vsa=%d", asa, vsa)
	}

	j := asa
	vb := vsa + j - asa
	if !pdwXsumchk(frame[vb]) {
		t.Fatalf("vector xsum fail at %d logical=0x%05X", vb, frame[vb]&0x1FFFFF)
	}
	vt := (frame[vb] >> 4) & 0x07
	if vt != 5 {
		t.Fatalf("expected alpha vector type 5, got %d", vt)
	}

	capcode := (frame[j] & 0x1FFFFF) - 32768
	if capcode != 1913 {
		t.Fatalf("capcode=%d want 1913", capcode)
	}

	w1 := int((frame[vb] >> 7) & 0x7F)
	w2 := int(((frame[vb]>>14)&0x7F)+int64(w1)) - 1
	frag := int((frame[w1] >> 11) & 0x03)
	t.Logf("capcode=%d frag=%d msg words %d..%d header=0x%05X",
		capcode, frag, w1, w2, frame[w1]&0x1FFFFF)

	msgs, _ := DecodeFromAudio(wav)
	if len(msgs) != 1 || msgs[0].Text != "HELLO WORLD" {
		t.Fatalf("decode: %+v", msgs)
	}
}
