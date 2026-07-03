package flex

import (
	"bytes"
	"encoding/binary"
	"math"
)

const (
	// Audio constants
	BaudRate1600  = 1600
	BaudRate3200  = 3200 // 4-FSK
	SampleRate    = 48000
	BitsPerSample = 16
	NumChannels   = 1
)

var (
	SymbolHigh = int16(-12287) // bit 1 (0xD001 as signed)
	SymbolLow  = int16(12287)  // bit 0 (0x2FFF as signed)
)

// ConvertToAudio converts raw data bytes into a baseband WAV (for standard 1600 baud 2-FSK)
func ConvertToAudio(data []byte) []byte {
	return ConvertToAudioWithBaudRate(data, BaudRate1600)
}

// ConvertToAudioWithBaudRate converts data bytes to WAV audio with specified baud rate.
// Uses baseband (DC levels): bit 1 = negative, bit 0 = positive.
func ConvertToAudioWithBaudRate(data []byte, baudRate int) []byte {
	samplesPerSymbol := float64(SampleRate) / float64(baudRate)
	numBits := len(data) * 8
	numSamples := int(float64(numBits) * samplesPerSymbol)

	audioData := make([]int16, numSamples)

	for byteIdx, b := range data {
		for bitPos := 7; bitPos >= 0; bitPos-- {
			bit := (b >> bitPos) & 1
			var sample int16

			if bit == 1 {
				sample = int16(SymbolHigh)
			} else {
				sample = int16(SymbolLow)
			}

			bitIndex := byteIdx*8 + (7 - bitPos)
			startIdx := int(math.Round(float64(bitIndex) * samplesPerSymbol))
			endIdx := int(math.Round(float64(bitIndex+1) * samplesPerSymbol))

			for j := startIdx; j < endIdx; j++ {
				audioData[j] = sample
			}
		}
	}

	return createWAVFile(audioData)
}

// FSK tone frequencies (mark=1, space=0)
const (
	FSKFreqSpace = 1600.0 // Hz, bit 0
	FSKFreqMark  = 3200.0 // Hz, bit 1
)

// ConvertToAudioFSK converts raw data bytes to FSK WAV audio (sine waves).
func ConvertToAudioFSK(data []byte, baudRate int) []byte {
	samplesPerSymbol := float64(SampleRate) / float64(baudRate)
	numBits := len(data) * 8
	numSamples := int(float64(numBits) * samplesPerSymbol)
	audioData := make([]int16, numSamples)

	const amplitude = 16000.0
	phase := 0.0

	for byteIdx, b := range data {
		for bitPos := 7; bitPos >= 0; bitPos-- {
			bit := (b >> bitPos) & 1
			freq := FSKFreqSpace
			if bit == 1 {
				freq = FSKFreqMark
			}
			phaseIncrement := 2.0 * math.Pi * freq / float64(SampleRate)

			bitIndex := byteIdx*8 + (7 - bitPos)
			startIdx := int(float64(bitIndex) * samplesPerSymbol)
			endIdx := int(float64(bitIndex+1) * samplesPerSymbol)

			for j := startIdx; j < endIdx; j++ {
				phase += phaseIncrement
				for phase > 2.0*math.Pi {
					phase -= 2.0 * math.Pi
				}
				audioData[j] = int16(amplitude * math.Sin(phase))
			}
		}
	}

	return createWAVFile(audioData)
}

func createWAVFile(samples []int16) []byte {
	var buf bytes.Buffer

	dataSize := uint32(len(samples) * 2)
	fileSize := 36 + dataSize
	byteRate := uint32(SampleRate * NumChannels * BitsPerSample / 8)
	blockAlign := uint16(NumChannels * BitsPerSample / 8)

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))            // chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // PCM format
	binary.Write(&buf, binary.LittleEndian, uint16(NumChannels))   // channels
	binary.Write(&buf, binary.LittleEndian, uint32(SampleRate))    // sample rate
	binary.Write(&buf, binary.LittleEndian, byteRate)              // byte rate
	binary.Write(&buf, binary.LittleEndian, blockAlign)            // block align
	binary.Write(&buf, binary.LittleEndian, uint16(BitsPerSample)) // bits per sample

	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)

	for _, sample := range samples {
		binary.Write(&buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}
