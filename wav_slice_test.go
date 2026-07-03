package flex

import (
	"os"
	"testing"
)

func TestReferenceWAVSlice(t *testing.T) {
	refData, err := os.ReadFile("tests/test_1600.wav")
	if err != nil {
		t.Skip("no reference wav")
	}
	sliced, err := sliceWAVPDWStyle(refData)
	if err != nil {
		t.Fatal(err)
	}
	frames, _ := DemodulateRawFrames(refData)
	t.Logf("reference: sliced %d bits, PLL frames=%d", len(sliced), len(frames))
	if len(frames) > 0 {
		t.Logf("  first frame cycle=%d frame=%d", frames[0].Cycle, frames[0].Frame)
	}
}

func TestBitstreamThroughWAVSlice(t *testing.T) {
	bits, err := BuildBitstream(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}, Mode1600_2, 3, 111)
	if err != nil {
		t.Fatal(err)
	}
	wav := modulateBits(bits, 1600)
	sliced, err := sliceWAVPDWStyle(wav)
	if err != nil {
		t.Fatal(err)
	}

	minLen := len(bits)
	if len(sliced) < minLen {
		minLen = len(sliced)
	}
	errors := 0
	for i := 0; i < minLen; i++ {
		if bits[i] != sliced[i] {
			errors++
		}
	}
	t.Logf("expected %d bits, sliced %d, errors %d (%.2f%%)",
		len(bits), len(sliced), errors, 100*float64(errors)/float64(minLen))

	frames, _ := DemodulateRawFrames(wav)
	if len(frames) > 0 {
		t.Logf("PLL demod: cycle=%d frame=%d cw0=0x%08X cw1=0x%08X cw2=0x%08X",
			frames[0].Cycle, frames[0].Frame,
			frames[0].Words[0], frames[0].Words[1], frames[0].Words[2])
	}
}

// sliceWAVPDWStyle approximates PDW sound_in.cpp threshold slicing on 16-bit WAV.
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
		samples[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
	}
	return samples, rate, nil
}
