package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	flex "github.com/sqpp/flex-golang"
)

func main() {
	outPath := flag.String("o", "", "Output WAV path (required)")
	mode := flag.String("mode", flex.Mode1600_2, "FLEX mode (currently only 1600/2)")
	capcode := flag.Int("capcode", 0, "Capcode (1..2097151)")
	text := flag.String("text", "", "Message text")
	msgType := flag.String("type", "alpha", "Message type: alpha, numeric, tone")
	cycle := flag.Int("cycle", 0, "FIW cycle field")
	frame := flag.Int("frame", 0, "FIW frame field")
	jsonPath := flag.String("json", "", "JSON file with a list of message objects")

	version := flag.Bool("v", false, "Show version information")
	versionLong := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *version || *versionLong {
		fmt.Println(flex.GetFullVersionInfo())
		os.Exit(0)
	}

	if *outPath == "" {
		fmt.Println("Usage: flex-encode -o <output.wav> --capcode <n> --text \"message\"")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var messages []flex.EncodeMessage
	if *jsonPath != "" {
		data, err := os.ReadFile(*jsonPath)
		if err != nil {
			log.Fatalf("Failed to read JSON: %v", err)
		}
		if err := json.Unmarshal(data, &messages); err != nil {
			log.Fatalf("Invalid JSON: %v", err)
		}
	} else {
		if *capcode == 0 {
			log.Fatal("Need --capcode and --text, or --json")
		}
		messages = []flex.EncodeMessage{{
			Capcode: *capcode,
			Type:    *msgType,
			Text:    *text,
		}}
	}

	wav, nBits, nSamples, err := flex.EncodeToWAVBytes(messages, *mode, *cycle, *frame)
	if err != nil {
		log.Fatalf("Encoding failed: %v", err)
	}
	if err := os.WriteFile(*outPath, wav, 0644); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	duration := float64(nSamples) / float64(flex.EncoderSampleRate)
	fmt.Printf("wrote %s: mode=%s %d message(s), %d bits, %d samples (%.3f s)\n",
		*outPath, *mode, len(messages), nBits, nSamples, duration)
}
