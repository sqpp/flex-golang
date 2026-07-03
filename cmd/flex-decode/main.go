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
	inputPath := flag.String("i", "", "Input WAV file (required)")
	inputPathLong := flag.String("input", "", "Input WAV file (required)")
	jsonOutput := flag.Bool("j", false, "JSON output")
	jsonOutputLong := flag.Bool("json", false, "JSON output")

	versionOutput := flag.Bool("v", false, "Show version information")
	versionOutputLong := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *versionOutput || *versionOutputLong {
		fmt.Println(flex.GetFullVersionInfo())
		os.Exit(0)
	}
	inputFile := *inputPath
	if *inputPathLong != "" {
		inputFile = *inputPathLong
	}

	isJson := *jsonOutput || *jsonOutputLong

	if inputFile == "" {
		fmt.Println("Usage: flex-decode -i <input.wav>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// Read file

	decodedMessages, err := flex.DecodeFromAudio(data)
	if err != nil {
		log.Fatalf("Decoding failed: %v", err)
	}

	if isJson {
		jsonMessages := make([]map[string]interface{}, len(decodedMessages))
		for i, msg := range decodedMessages {
			msgType := "unknown"
			if msg.IsNumeric {
				msgType = "numeric"
			} else if msg.Type == flex.PageAlphanumeric {
				msgType = "alphanumeric"
			} else if msg.Type == flex.PageTone {
				msgType = "tone"
			}

			jsonMessages[i] = map[string]interface{}{
				"address":  msg.Capcode,
				"function": int(msg.Type),
				"message":  msg.Text,
				"type":     msgType,
			}
		}
		
		baudRate := 1600
		levels := 2
		if len(decodedMessages) > 0 {
			baudRate = decodedMessages[0].Baud
			levels = decodedMessages[0].Levels
		}
		bitRate := baudRate
		if levels == 4 {
			bitRate *= 2
		}

		result := map[string]interface{}{
			"success":  true,
			"messages": jsonMessages,
			"baud":     bitRate,
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		baudRate := 1600
		levels := 2
		if len(decodedMessages) > 0 {
			baudRate = decodedMessages[0].Baud
			levels = decodedMessages[0].Levels
		}
		bitRate := baudRate
		if levels == 4 {
			bitRate *= 2
		}
		
		baudStr := fmt.Sprintf("FLEX-%d/%d", bitRate, levels)
		fmt.Printf("%s: Decoded messages:\n", baudStr)
		for _, msg := range decodedMessages {
			msgType := "ALPHA"
			if msg.IsNumeric {
				msgType = "NUMERIC"
			}
			fmt.Printf("Address: %07d  Function: %d  %-7s  Message: %s\n",
				msg.Capcode, int(msg.Type), msgType, msg.Text)
		}
	}
}
