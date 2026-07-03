# FLEX-GO v1.0.0

A complete Go implementation of the Motorola FLEX pager protocol — encoder, decoder, and everything in between. CLI layout mirrors [`pocsag-golang`](https://github.com/sqpp/pocsag-golang).

## What it can do

- Encode FLEX messages as WAV audio (1600/2, 1600/4, 3200/2, 3200/4, 6400/4)
- Decode those WAV files back to text with a native PLL demodulator
- BCH(31,21) codewords with Motorola 4-bit checksum validation
- Capcode addressing with alphanumeric, numeric, and tone page types
- Frame Information Words (FIW) and Block Information Words (BIW)
- JSON output for scripting and API integration
- Pure Go — no CGO or external demod tools required

---

## Installation

```bash
# Encoder
go install github.com/sqpp/flex-golang/cmd/flex-encode@latest

# Decoder
go install github.com/sqpp/flex-golang/cmd/flex-decode@latest
```

Or build from source:

```bash
git clone https://github.com/sqpp/flex-golang.git
cd flex-golang
make build          # Linux/macOS
make.bat build      # Windows
# Binaries land in: bin/flex-encode, bin/flex-decode
```

---

## Encoder (`flex-encode`)

Generate a FLEX message as a WAV file.

**Required (unless using `--reference` or `--messages`):**
- `-a` / `--address` — pager address / capcode (1..2097151)
- `-m` / `--message` — the message text

**Optional:**
- `-o` / `--output` — output WAV file (default: `output.wav`)
- `-f` / `--function` — `2` = tone, `3` = numeric, `5` = alphanumeric (default: `5`)
- `--mode` — FLEX mode: `1600/2`, `1600/4`, `3200/2`, `3200/4`, `6400/4` (default: `1600/2`)
- `--cycle` / `--frame` — FIW cycle and frame fields (optional)
- `--reference` — encode the known-good PDW reference page from `tests/test_1600.wav`
- `--messages` / `--messages-file` — JSON file with a list of message objects
- `-j` / `--json` — print result as JSON instead of human-readable text
- `-v` / `--version` — show version info

**Examples:**

```bash
# Basic message
flex-encode -a 1913 -m "HELLO WORLD" -o message.wav

# Long flags
flex-encode --address 1913 --message "HELLO WORLD" --output message.wav

# Reference page (cap 1913, cycle 3, frame 111)
flex-encode -o reference.wav --reference

# 3200/4 mode
flex-encode -a 1913 -m "FAST MSG" --mode 3200/4 -o fast.wav

# JSON output (great for scripts)
flex-encode -a 1913 -m "TEST" -o test.wav --json
```

**Normal output:**
```
✅ Generated message.wav
   Address: 1913, Function: 5, Baud: 1600, Message: HELLO WORLD
   Size: 220104 bytes, Duration: 2.50 s

Decode: flex-decode -i message.wav
```

**JSON output:**
```json
{
  "success": true,
  "output": "message.wav",
  "address": 1913,
  "function": 5,
  "message": "HELLO WORLD",
  "baud": 1600,
  "type": "alphanumeric",
  "size": 220104,
  "duration_s": 2.495
}
```

---

## Decoder (`flex-decode`)

Decode a FLEX WAV back to text.

**Options:**
- `-i` / `--input` — input WAV file (required)
- `--no-tones` — filter out tone-only messages
- `-j` / `--json` — JSON output
- `-v` / `--version` — show version info

**Examples:**

```bash
flex-decode -i message.wav
flex-decode -i message.wav --json
flex-decode --version
```

**Normal output:**
```
FLEX-1600/2: Decoded messages:
Address: 0001913  Function: 5  ALPHA    Message: HELLO WORLD
```

**JSON output:**
```json
{
  "success": true,
  "baud": 1600,
  "messages": [
    {
      "address": 1913,
      "function": 5,
      "message": "HELLO WORLD",
      "type": "alphanumeric"
    }
  ]
}
```

---

## Using as a Go library

**Encode and write a WAV:**
```go
import flex "github.com/sqpp/flex-golang"

msg := flex.EncodeMessage{Capcode: 1913, Type: "alpha", Text: "HELLO WORLD"}
wav, _, _, err := flex.EncodeToWAVBytes([]flex.EncodeMessage{msg}, flex.Mode1600_2, 0, 0)
os.WriteFile("output.wav", wav, 0644)
```

**Decode a WAV:**
```go
wavData, _ := os.ReadFile("message.wav")
messages, err := flex.DecodeFromAudio(wavData)
for _, msg := range messages {
    fmt.Printf("Message to %07d: %s\n", msg.Capcode, msg.Text)
}
```

**Key functions:**

| Function | Description |
|----------|-------------|
| `EncodeToWAVBytes(messages, mode, cycle, frame)` | Encode messages to WAV bytes |
| `EncodeToWAVFile(messages, path, mode, cycle, frame)` | Encode and write a WAV file |
| `DecodeFromAudio(wavData)` | Decode a WAV into messages |
| `DemodulateRawFrames(wavData)` | Demodulate to raw phase codewords |
| `EncodeModeNames()` | List supported encoder modes |
| `EncodeModeBitRate(mode)` | On-air bit rate for a mode name |

---

## Testing

```bash
go test -v ./...
```

Tests live in the `tests/` directory. Reference captures (`test_1600.wav`, `test_3200.wav`, `test_6400.wav`) are included for roundtrip and decode validation.

---

## About addresses

FLEX addresses are capcodes in the range 1..2097151. The encoder and decoder use the full capcode value. Function codes follow the FLEX vector type field (`2` = tone, `3` = numeric, `5` = alphanumeric).

---

## Credits

Part of [PagerCast](https://pagercast.com). Decoder logic references multimon-ng `demod_flex.c` and PDW `Flex.cpp`.

## License

BSD-2-Clause — see [LICENSE](LICENSE).
