package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestFlexCodewordRoundtrip(t *testing.T) {
	for _, logical := range []uint32{
		flex.ExportBuildFIWData(0, 0),
		flex.ExportBuildFIWData(5, 42),
	} {
		cw := flex.ExportEncodeWord(logical)
		got, errs := flex.FLEXBCHDecode32(cw)
		if got != logical || errs != 0 || !flex.ExportFLEXChecksum(got) {
			t.Fatalf("logical=0x%X cw=0x%08X got=0x%X errs=%d", logical, cw, got, errs)
		}
	}
}

func TestEncodeToWAV(t *testing.T) {
	msg := flex.EncodeMessage{Capcode: 1913, Type: "alpha", Text: "HELLO WORLD"}
	wav, nBits, nSamples, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{msg}, flex.Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || nBits == 0 || nSamples == 0 {
		t.Fatalf("wav=%d bits=%d samples=%d", len(wav), nBits, nSamples)
	}
}

func TestDecodeReferenceWAV(t *testing.T) {
	data, err := os.ReadFile("./test_6400.wav")
	if err != nil {
		t.Skip("./test_6400.wav not available")
	}
	msgs, err := flex.DecodeFromAudio(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Fatal("reference wav produced no messages")
	}
	t.Logf("reference decoded %d messages", len(msgs))
}
