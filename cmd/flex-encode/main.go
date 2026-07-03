package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	flex "github.com/sqpp/flex-golang"
)

func main() {
	address := flag.Int("address", 0, "Pager address (capcode) - REQUIRED")
	flag.IntVar(address, "a", 0, "Pager address (capcode) - REQUIRED")

	message := flag.String("message", "", "Message text to send - REQUIRED")
	flag.StringVar(message, "m", "", "Message text to send - REQUIRED")

	output := flag.String("output", "output.wav", "Output WAV file path")
	flag.StringVar(output, "o", "output.wav", "Output WAV file path")

	funcCode := flag.Uint("function", uint(flex.PageAlphanumeric), "Message type: 2=tone, 3=numeric, 5=alphanumeric (default: 5)")
	flag.UintVar(funcCode, "f", uint(flex.PageAlphanumeric), "Message type: 2=tone, 3=numeric, 5=alphanumeric")

	mode := flag.String("mode", flex.Mode1600_2, fmt.Sprintf("FLEX mode: %s (default: %s)", strings.Join(flex.EncodeModeNames(), ", "), flex.Mode1600_2))

	cycle := flag.Int("cycle", 0, "FIW cycle field (optional)")
	frame := flag.Int("frame", 0, "FIW frame field (optional)")

	reference := flag.Bool("reference", false, "Encode the known-good PDW message from tests/test_1600.wav (cap 1913)")

	messagesFile := flag.String("messages", "", "JSON file with a list of message objects (optional)")
	flag.StringVar(messagesFile, "messages-file", "", "JSON file with a list of message objects (optional)")

	jsonOutput := flag.Bool("json", false, "Output result as JSON")
	flag.BoolVar(jsonOutput, "j", false, "Output result as JSON")

	version := flag.Bool("version", false, "Show version information")
	flag.BoolVar(version, "v", false, "Show version information")

	flag.Parse()

	if *version {
		fmt.Println(flex.GetFullVersionInfo())
		os.Exit(0)
	}

	baud, err := flex.EncodeModeBitRate(*mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var messages []flex.EncodeMessage
	cycleVal, frameVal := *cycle, *frame

	switch {
	case *reference:
		messages = []flex.EncodeMessage{{
			Capcode: flex.ReferenceCapcode1913,
			Type:    "alpha",
			Text:    flex.ReferenceMessage1913,
		}}
		if cycleVal == 0 && frameVal == 0 {
			cycleVal = flex.ReferenceCycle1913
			frameVal = flex.ReferenceFrame1913
		}
	case *messagesFile != "":
		data, err := os.ReadFile(*messagesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading messages file: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &messages); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing messages file: %v\n", err)
			os.Exit(1)
		}
	default:
		if *address == 0 || *message == "" {
			fmt.Fprintln(os.Stderr, "Error: Address and message are required")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Note: FLEX addresses (capcodes) are in the range 1..2097151")
			fmt.Fprintln(os.Stderr, "\nUsage examples:")
			fmt.Fprintln(os.Stderr, "  flex-encode --address 1913 --message \"HELLO WORLD\" --output test.wav")
			fmt.Fprintln(os.Stderr, "  flex-encode -a 1913 -m \"HELLO WORLD\" -o test.wav")
			fmt.Fprintln(os.Stderr, "  flex-encode -o test.wav --reference")
			fmt.Fprintln(os.Stderr, "  flex-encode -o test.wav --messages messages.json")
			fmt.Fprintln(os.Stderr, "")
			flag.Usage()
			os.Exit(1)
		}
		msgType, err := flex.EncodeTypeFromFunction(int(*funcCode))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		messages = []flex.EncodeMessage{{
			Capcode: *address,
			Type:    msgType,
			Text:    *message,
		}}
	}

	if len(messages) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No messages to encode")
		os.Exit(1)
	}

	wav, _, nSamples, err := flex.EncodeToWAVBytes(messages, *mode, cycleVal, frameVal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, wav, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WAV file: %v\n", err)
		os.Exit(1)
	}

	durationSec := float64(nSamples) / float64(flex.EncoderSampleRate)

	if *jsonOutput {
		result := map[string]interface{}{
			"success":    true,
			"output":     *output,
			"baud":       baud,
			"size":       len(wav),
			"duration_s": durationSec,
		}
		if len(messages) == 1 {
			result["address"] = messages[0].Capcode
			result["function"] = functionForMessage(messages[0].Type)
			result["message"] = messages[0].Text
			result["type"] = flex.EncodeTypeLabel(messages[0].Type)
		}
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("✅ Generated %s\n", *output)
		if len(messages) == 1 {
			fmt.Printf("   Address: %d, Function: %d, Baud: %d, Message: %s\n",
				messages[0].Capcode, functionForMessage(messages[0].Type), baud, messages[0].Text)
		} else {
			fmt.Printf("   Messages: %d, Baud: %d\n", len(messages), baud)
		}
		if cycleVal != 0 || frameVal != 0 {
			fmt.Printf("   Cycle: %d, Frame: %d\n", cycleVal, frameVal)
		}
		fmt.Printf("   Size: %d bytes, Duration: %.2f s\n", len(wav), durationSec)
		fmt.Printf("\nDecode: flex-decode -i %s\n", *output)
	}
}

func functionForMessage(typeName string) int {
	switch typeName {
	case "numeric":
		return int(flex.PageStdNumeric)
	case "tone":
		return int(flex.PageTone)
	default:
		return int(flex.PageAlphanumeric)
	}
}
