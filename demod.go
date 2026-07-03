package flex

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const FrameSyncWord = 0x9c9acf1e
const FrameSyncWordInverted = 0x636530e1
const IdleBlockWord = 0x00000000
const IdleBlockWord2 = 0xFFFFFFFF

// DemodulateRawFrames returns MSB-assembled phase-A codewords for each synced frame.
func DemodulateRawFrames(wavData []byte) ([]RawPhaseFrame, error) {
	dataOffset := bytes.Index(wavData, []byte("data"))
	startIdx := 44
	if dataOffset != -1 {
		startIdx = dataOffset + 8
	}
	var sampleRate uint32 = 48000
	if len(wavData) > 28 {
		sampleRate = binary.LittleEndian.Uint32(wavData[24:28])
	}
	d := NewDemodulator(sampleRate)
	lpfBuf := make([]float32, 8)
	lpfIdx := 0
	var lpfSum float32
	for i := startIdx; i < len(wavData)-1; i += 2 {
		rawSample := float32(int16(binary.LittleEndian.Uint16(wavData[i:])))
		lpfSum -= lpfBuf[lpfIdx]
		lpfBuf[lpfIdx] = rawSample
		lpfSum += rawSample
		lpfIdx = (lpfIdx + 1) % len(lpfBuf)
		sample := lpfSum / float32(len(lpfBuf))
		d.PushSample(sample)
	}
	return d.capturedFrames, nil
}

// DecodeFromAudio decodes FLEX messages from a 1600 baud 2-FSK WAV file
func DecodeFromAudio(wavData []byte) ([]Message, error) {
	return DecodeFromAudioWithBaudRate(wavData, BaudRate1600)
}

// DecodeFromAudioWithBaudRate decodes FLEX from WAV audio data with specified baud rate.
// For FLEX, baud rate is technically auto-detected from the Sync-1 code, but this
// parameter is kept for signature compatibility.
func DecodeFromAudioWithBaudRate(wavData []byte, baudRate int) ([]Message, error) {
	dataOffset := bytes.Index(wavData, []byte("data"))
	startIdx := 44
	if dataOffset != -1 {
		startIdx = dataOffset + 8
	}

	var sampleRate uint32 = 48000
	if len(wavData) > 28 {
		sampleRate = binary.LittleEndian.Uint32(wavData[24:28])
	}

	demod := NewDemodulator(sampleRate)
	var allMessages []Message

	// Simple low-pass filter to remove high-frequency noise (acts like sox anti-aliasing)
	// 44100Hz / 8 = 5512Hz cutoff roughly
	lpfBuf := make([]float32, 8)
	lpfIdx := 0
	var lpfSum float32

	for i := startIdx; i < len(wavData)-1; i += 2 {
		rawSample := float32(int16(binary.LittleEndian.Uint16(wavData[i:])))
		
		lpfSum -= lpfBuf[lpfIdx]
		lpfBuf[lpfIdx] = rawSample
		lpfSum += rawSample
		lpfIdx = (lpfIdx + 1) % len(lpfBuf)
		
		sample := lpfSum / float32(len(lpfBuf))

		msgs := demod.PushSample(sample)
		if len(msgs) > 0 {
			for _, msg := range msgs {
				if msg.Valid && !containsMsg(allMessages, msg) {
					allMessages = append(allMessages, msg)
				}
			}
		}
	}

	return allMessages, nil
}

func containsMsg(msgs []Message, m Message) bool {
	for _, existing := range msgs {
		if existing.Capcode == m.Capcode && existing.Text == m.Text {
			return true
		}
	}
	return false
}

// DecodeFromBitstream decodes FLEX messages from a raw bitstream.
// Warning: Deprecated for FLEX since the native PLL requires raw audio samples.
func DecodeFromBitstream(bits []byte) ([]Message, error) {
	return nil, fmt.Errorf("DecodeFromBitstream is no longer supported; use DecodeFromAudio with native PLL")
}
