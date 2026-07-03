package flex

import "math"

const (
	// EncoderSampleRate matches FLEX reference captures (44100 Hz).
	EncoderSampleRate = 44100
	// PDW: high audio = mark (bit 1), low audio = space (bit 0).
	flexMarkLevel  int16 = 21400
	flexSpaceLevel int16 = -21400
)

// modulateBits converts a FLEX bitstream to 16-bit mono baseband WAV for PDW.
func modulateBits(bits []byte, baudRate int) []byte {
	sampleRate := float64(EncoderSampleRate)
	samplesPerSymbol := sampleRate / float64(baudRate)
	numSamples := int(math.Ceil(float64(len(bits)) * samplesPerSymbol))
	audio := make([]int16, numSamples)

	for bitIdx, bit := range bits {
		var sample int16
		if bit != 0 {
			sample = flexMarkLevel
		} else {
			sample = flexSpaceLevel
		}

		startIdx := int(math.Round(float64(bitIdx) * samplesPerSymbol))
		endIdx := int(math.Round(float64(bitIdx+1) * samplesPerSymbol))
		if endIdx > numSamples {
			endIdx = numSamples
		}
		for j := startIdx; j < endIdx; j++ {
			audio[j] = sample
		}
	}

	return createWAVFileAtRate(audio, EncoderSampleRate)
}
