package flex_test

import (
	"os"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestReferenceWAVSlice(t *testing.T) {
	refData, err := os.ReadFile("./test_1600.wav")
	if err != nil {
		t.Skip("no reference wav")
	}
	sliced, err := sliceWAVPDWStyle(refData)
	if err != nil {
		t.Fatal(err)
	}
	frames, _ := flex.DemodulateRawFrames(refData)
	t.Logf("reference: sliced %d bits, PLL frames=%d", len(sliced), len(frames))
	if len(frames) > 0 {
		t.Logf("  first frame cycle=%d frame=%d", frames[0].Cycle, frames[0].Frame)
	}
}

func TestBitstreamThroughWAVSlice(t *testing.T) {
	wav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}}, flex.Mode1600_2, 3, 111)
	if err != nil {
		t.Fatal(err)
	}

	frames, _ := flex.DemodulateRawFrames(wav)
	if len(frames) == 0 {
		t.Fatal("PLL demod produced no frames")
	}
	t.Logf("PLL demod: cycle=%d frame=%d cw0=0x%08X cw1=0x%08X cw2=0x%08X",
		frames[0].Cycle, frames[0].Frame,
		frames[0].Words[0], frames[0].Words[1], frames[0].Words[2])
}

func sliceWAVPDWStyle(wav []byte) ([]byte, error) {
	samples, rate, err := readWAVSamples(wav)
	if err != nil {
		return nil, err
	}
	spb := float64(rate) / 1600.0
	var bits []byte
	center := 0.0
	bit := byte(0)
	const thresh = 2000.0

	for sym := 0; ; sym++ {
		idx := int(float64(sym)*spb + spb*0.5)
		if idx >= len(samples) {
			break
		}
		val := float64(samples[idx])
		center = center*0.999 + val*0.001

		if val > center+thresh && bit == 0 {
			bit = 1
		} else if val < center-thresh && bit == 1 {
			bit = 0
		}
		bits = append(bits, bit)
	}
	return bits, nil
}

func readWAVSamples(wav []byte) ([]int16, int, error) {
	if len(wav) < 44 {
		return nil, 0, nil
	}
	rate := int(wav[24]) | int(wav[25])<<8 | int(wav[26])<<16 | int(wav[27])<<24
	dataSize := int(wav[40]) | int(wav[41])<<8 | int(wav[42])<<16 | int(wav[43])<<24
	data := wav[44:]
	if len(data) > dataSize {
		data = data[:dataSize]
	}
	n := len(data) / 2
	samples := make([]int16, n)
	for i := 0; i < n; i++ {
		samples[i] = int16(data[i*2]) | int16(data[i+1])<<8
	}
	return samples, rate, nil
}
