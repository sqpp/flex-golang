package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestEncodeDecodeAllModes(t *testing.T) {
	msg := flex.EncodeMessage{Capcode: 123456, Type: "alpha", Text: "MODE TEST"}

	for _, modeName := range flex.EncodeModeNames() {
		t.Run(modeName, func(t *testing.T) {
			wav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{msg}, modeName, 0, 42)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := flex.DecodeFromAudio(wav)
			if err != nil {
				t.Fatal(err)
			}
			if len(decoded) == 0 {
				t.Fatalf("%s: no messages decoded", modeName)
			}
			found := false
			for _, m := range decoded {
				if m.Capcode == 123456 && m.Text == "MODE TEST" {
					found = true
					if m.Baud == 0 {
						t.Logf("baud not set on message")
					}
					t.Logf("%s: cap=%d baud=%d levels=%d phase=%c text=%q",
						modeName, m.Capcode, m.Baud, m.Levels, m.Phase, m.Text)
				}
			}
			if !found {
				t.Fatalf("%s: cap 123456 not found in %+v", modeName, decoded)
			}
		})
	}
}

func TestEncodeReferenceAllModesWAV(t *testing.T) {
	msg := flex.EncodeMessage{
		Capcode: flex.ReferenceCapcode1913,
		Type:    "alpha",
		Text:    flex.ReferenceMessage1913,
	}
	for _, modeName := range flex.EncodeModeNames() {
		wav, nSym, nSamples, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{msg}, modeName, flex.ReferenceCycle1913, flex.ReferenceFrame1913)
		if err != nil {
			t.Fatalf("%s: %v", modeName, err)
		}
		if len(wav) < 44 || nSym == 0 || nSamples == 0 {
			t.Fatalf("%s: bad output len=%d sym=%d samples=%d", modeName, len(wav), nSym, nSamples)
		}
	}
}

func TestReferenceWAVFilesStillDecode(t *testing.T) {
	files := map[string]string{
		"1600": "./test_1600.wav",
		"3200": "./test_3200.wav",
		"6400": "./test_6400.wav",
	}
	for label, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Logf("skip %s: %v", label, err)
			continue
		}
		msgs, err := flex.DecodeFromAudio(data)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		t.Logf("%s: decoded %d messages", label, len(msgs))
		if len(msgs) == 0 {
			t.Fatalf("%s: expected messages", label)
		}
	}
}
