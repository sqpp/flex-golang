package flex

import (
	"bytes"
	"encoding/binary"
	"math"
)

const (
	// EncoderSampleRate matches FLEX reference captures (44100 Hz).
	EncoderSampleRate = 44100
	// PDW: high audio = mark (bit 1), low audio = space (bit 0).
	flexMarkLevel  int16 = 21400
	flexSpaceLevel int16 = -21400
	// Inner 4-level steps sit between mark/space for the native demodulator.
	flex4LevelNegInner int16 = -10000
	flex4LevelPosInner int16 = 10000
)

// flexTransmission holds header (1600 baud), data symbols, and trailer (1600 baud).
type flexTransmission struct {
	header   []byte
	data     []byte
	trailer  []byte
	dataBaud int
	levels   int
}

func symbolCount(tx flexTransmission) int {
	return len(tx.header) + len(tx.data) + len(tx.trailer)
}

// modulateTransmission converts a FLEX frame to 16-bit mono baseband WAV.
func modulateTransmission(tx flexTransmission) []byte {
	sampleRate := float64(EncoderSampleRate)
	headerSPB := sampleRate / 1600.0
	dataSPB := sampleRate / float64(tx.dataBaud)

	headerSamples := int(math.Ceil(float64(len(tx.header)) * headerSPB))
	dataSamples := int(math.Ceil(float64(len(tx.data)) * dataSPB))
	trailerSamples := int(math.Ceil(float64(len(tx.trailer)) * headerSPB))
	audio := make([]int16, headerSamples+dataSamples+trailerSamples)

	write2Level := func(offset int, spb float64, symbols []byte) {
		for i, sym := range symbols {
			sample := flexSpaceLevel
			if sym&1 != 0 {
				sample = flexMarkLevel
			}
			start := offset + int(math.Round(float64(i)*spb))
			end := offset + int(math.Round(float64(i+1)*spb))
			if end > len(audio) {
				end = len(audio)
			}
			for j := start; j < end; j++ {
				audio[j] = sample
			}
		}
	}

	write4Level := func(offset int, spb float64, symbols []byte) {
		levels := []int16{flexSpaceLevel, flex4LevelNegInner, flex4LevelPosInner, flexMarkLevel}
		for i, sym := range symbols {
			idx := int(sym) & 3
			sample := levels[idx]
			start := offset + int(math.Round(float64(i)*spb))
			end := offset + int(math.Round(float64(i+1)*spb))
			if end > len(audio) {
				end = len(audio)
			}
			for j := start; j < end; j++ {
				audio[j] = sample
			}
		}
	}

	write2Level(0, headerSPB, tx.header)
	dataOffset := headerSamples
	if tx.levels == 2 {
		write2Level(dataOffset, dataSPB, tx.data)
	} else {
		write4Level(dataOffset, dataSPB, tx.data)
	}
	write2Level(dataOffset+dataSamples, headerSPB, tx.trailer)

	return createWAVFileAtRate(audio, EncoderSampleRate)
}

// modulateBits converts a FLEX bitstream to 16-bit mono baseband WAV for PDW.
// Deprecated: use modulateTransmission via EncodeToWAVBytes.
func modulateBits(bits []byte, baudRate int) []byte {
	return modulateTransmission(flexTransmission{
		header:   bits,
		data:     nil,
		dataBaud: baudRate,
		levels:   2,
	})
}

func createWAVFileAtRate(samples []int16, sampleRate uint32) []byte {
	const (
		numChannels   = 1
		bitsPerSample = 16
	)
	var buf bytes.Buffer

	dataSize := uint32(len(samples) * 2)
	fileSize := 36 + dataSize
	byteRate := uint32(sampleRate * numChannels * bitsPerSample / 8)
	blockAlign := uint16(numChannels * bitsPerSample / 8)

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(numChannels))
	binary.Write(&buf, binary.LittleEndian, sampleRate)
	binary.Write(&buf, binary.LittleEndian, byteRate)
	binary.Write(&buf, binary.LittleEndian, blockAlign)
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)

	for _, sample := range samples {
		binary.Write(&buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}
