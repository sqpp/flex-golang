package flex

import (
	"os"
	"testing"
)

func TestFlexCodewordRoundtrip(t *testing.T) {
	for _, logical := range []uint32{
		buildFIWData(0, 0),
		buildFIWData(5, 42),
	} {
		cw := encodeWord(logical)
		got, errs := FLEXBCHDecode32(cw)
		if got != logical || errs != 0 || !FLEXChecksum(got) {
			t.Fatalf("logical=0x%X cw=0x%08X got=0x%X errs=%d", logical, cw, got, errs)
		}
	}
}

func TestEncodeToWAV(t *testing.T) {
	msg := EncodeMessage{Capcode: 1913, Type: "alpha", Text: "HELLO WORLD"}
	wav, nBits, nSamples, err := EncodeToWAVBytes([]EncodeMessage{msg}, Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || nBits == 0 || nSamples == 0 {
		t.Fatalf("wav=%d bits=%d samples=%d", len(wav), nBits, nSamples)
	}
}

func TestDecodeReferenceWAV(t *testing.T) {
	data, err := os.ReadFile("tests/test_6400.wav")
	if err != nil {
		t.Skip("tests/test_6400.wav not available")
	}
	msgs, err := DecodeFromAudio(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Fatal("reference wav produced no messages")
	}
	t.Logf("reference decoded %d messages", len(msgs))
}
