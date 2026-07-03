package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	flex "github.com/sqpp/flex-golang"
)

func main() {
	inputFile := flag.String("input", "", "Input WAV file to decode (required)")
	flag.StringVar(inputFile, "i", "", "Input WAV file to decode (required)")

	jsonOutput := flag.Bool("json", false, "Output result as JSON")
	flag.BoolVar(jsonOutput, "j", false, "Output result as JSON")

	version := flag.Bool("version", false, "Show version information")
	flag.BoolVar(version, "v", false, "Show version information")

	noTones := flag.Bool("no-tones", false, "Filter out tone-only messages")

	flag.Parse()

	if *version {
		fmt.Println(flex.GetFullVersionInfo())
		os.Exit(0)
	}

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: Input file required")
		fmt.Fprintln(os.Stderr, "\nUsage examples:")
		fmt.Fprintln(os.Stderr, "  flex-decode --input message.wav")
		fmt.Fprintln(os.Stderr, "  flex-decode -i message.wav")
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	decodedMessages, err := flex.DecodeFromAudio(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding: %v\n", err)
		os.Exit(1)
	}

	if len(decodedMessages) == 0 {
		if *jsonOutput {
			result := map[string]interface{}{
				"success":  true,
				"messages": []interface{}{},
			}
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println("No messages found")
		}
		return
	}

	if *jsonOutput {
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

			outText := msg.Text
			if msg.Type == flex.PageAlphanumeric && msg.Frag != 3 {
				fragNum := (3 - msg.Frag) + 1
				outText = fmt.Sprintf("[Continued message - Fragment #%d] %s", fragNum, msg.Text)
			}

			jsonMessages[i] = map[string]interface{}{
				"address":  msg.Capcode,
				"function": int(msg.Type),
				"message":  outText,
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
			if *noTones && msg.Type == flex.PageTone {
				continue
			}

			msgType := "ALPHA"
			if msg.Type == flex.PageTone {
				msgType = "TONE"
			} else if msg.IsNumeric {
				msgType = "NUMERIC"
			}

			outText := msg.Text
			if msg.Type == flex.PageAlphanumeric && msg.Frag != 3 {
				fragNum := (3 - msg.Frag) + 1
				outText = fmt.Sprintf("[Continued message - Fragment #%d] %s", fragNum, msg.Text)
			}

			fmt.Printf("Address: %07d  Function: %d  %-7s  Message: %s\n",
				msg.Capcode, int(msg.Type), msgType, outText)
		}
	}
}
