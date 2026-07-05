# Speech-to-Text (STT)

DietDaemon accepts voice messages (audio) on any supported messaging adapter. When
`ENABLE_STT=true`, audio is transcribed to text by a [whisper.cpp](https://github.com/ggerganov/whisper.cpp)
server before it enters the parser pipeline. The parser never knows the input was audio —
STT and parser tier are independent knobs.

## How it works

```
Voice message → [Messaging Adapter] → [whisper.cpp] → transcript text
                                                          ↓
                                                  [Parser pipeline]
                                                  (PARSER_TIER=0,1,2)
```

1. User sends a voice/audio message to the bot (Telegram, Discord, or Matrix).
2. DietDaemon uploads the raw audio bytes to the whisper.cpp `/inference` endpoint.
3. The transcript text replaces the message body; the detected language (BCP-47 locale
   hint, e.g. `en`, `pt`, `de`) is attached to the message.
4. The transcript then flows through the normal parser pipeline at whatever `PARSER_TIER`
   is configured — including Tier 0 (deterministic, no models).

## Prerequisites

- A running **whisper.cpp** server with HTTP API enabled. Build it from source:
  ```bash
  git clone https://github.com/ggerganov/whisper.cpp
  cd whisper.cpp
  cmake -B build
  cmake --build build --config Release
  # Download a model (tiny is the smallest, large-v3 is most accurate)
  bash models/download-ggml-model.sh base
  # Start the HTTP server
  ./build/bin/whisper-server -m models/ggml-base.bin --host 0.0.0.0 --port 8080
  ```
  See the [whisper.cpp docs](https://github.com/ggerganov/whisper.cpp/tree/master/examples/server)
  for GPU acceleration, model selection, and tuning options.

- **GPU strongly recommended** for reasonable latency. CPU transcription of a 5-second
  clip with the `base` model takes ~2-3 seconds on a modern CPU; `tiny` is faster but
  less accurate. With GPU (CUDA/Metal), it's near-instant.

## Configuration

```bash
# .env
ENABLE_STT=true
WHISPER_URL=http://whisper:8080          # whisper.cpp server address
```

| Variable       | Default                  | Description                              |
|----------------|--------------------------|------------------------------------------|
| `ENABLE_STT`   | `false`                  | Enable speech-to-text for audio messages |
| `WHISPER_URL`  | `http://whisper:8080`    | whisper.cpp HTTP server base URL         |

When `ENABLE_STT=false` (default), audio messages receive a prompt asking the user to
send text instead. No whisper server is needed.

## Docker Compose

Add a whisper.cpp sidecar to `docker-compose.yml`:

```yaml
services:
  whisper:
    image: ghcr.io/ggerganov/whisper.cpp:latest
    container_name: dietdaemon-whisper
    restart: unless-stopped
    command:
      - "-m"
      - "/models/ggml-base.bin"
      - "--host"
      - "0.0.0.0"
      - "--port"
      - "8080"
    volumes:
      - ./whisper-models:/models
    networks:
      - dokploy-network

  # In the dietdaemon service, add:
  #   environment:
  #     ENABLE_STT: "true"
  #     WHISPER_URL: http://whisper:8080
```

Download a model into `./whisper-models/` before starting:
```bash
mkdir -p whisper-models
wget -O whisper-models/ggml-base.bin \
  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin
```

## Tier independence

STT and `PARSER_TIER` are independent. Either can be toggled without affecting the other:

| `ENABLE_STT` | `PARSER_TIER` | Behaviour                                          |
|--------------|---------------|----------------------------------------------------|
| `false`      | `0`           | Text only, deterministic parse (default)           |
| `false`      | `1` or `2`    | Text only, AI-powered parse                        |
| `true`       | `0`           | Voice → deterministic parse (no LLM needed)        |
| `true`       | `1` or `2`    | Voice → AI-powered parse (full stack)              |

## Language detection

whisper.cpp returns a BCP-47 language code (e.g. `en`, `pt`, `de`) in its response.
DietDaemon attaches this as the message locale. The parser uses this locale hint to
choose language-specific tokenization rules (e.g. Portuguese "colher" vs English
"spoon").

If whisper returns no language code (empty string), the existing locale from the
messaging adapter (Telegram user language, Matrix room locale, etc.) is preserved.

## Error behaviour

| Scenario                          | User sees                                                |
|-----------------------------------|----------------------------------------------------------|
| STT disabled (`ENABLE_STT=false`) | "Audio messages are not supported (STT is disabled)..."  |
| whisper server unreachable        | "Couldn't transcribe audio: ... Try sending as text."    |
| whisper returns HTTP error        | "Couldn't transcribe audio: status 500: ..."             |
| Transcription returned empty text | "Couldn't understand the audio. Try speaking clearly..." |
| Transcription succeeded           | Normal meal reply with macros                            |

**No audio data is stored.** The raw audio bytes are sent to whisper, the transcript is
stored with the meal in the database, and the audio payload is discarded. No audio files
are written to disk.

## Troubleshooting

### "Couldn't transcribe audio: whisper: inference: connection refused"

whisper.cpp server is not running or `WHISPER_URL` points to the wrong host/port.
Verify with:
```bash
curl http://whisper:8080/inference -F "file=@test.wav"
```

### Transcription is inaccurate (wrong words, garbage output)

- Use a larger model (`small`, `medium`, or `large-v3` instead of `base`/`tiny`).
- Ensure the audio format is WAV. whisper.cpp's HTTP server expects WAV by default.
- Background noise or multiple speakers degrade accuracy. Try speaking closer to the
  mic in a quiet environment.

### Transcription is slow (5+ seconds for a short clip)

- CPU inference is slow. Use a GPU build (`WHISPER_CUBLAS=1` for NVIDIA,
  `WHISPER_METAL=1` for macOS).
- Drop to a smaller model (`tiny` is ~1 GB, `base` is ~150 MB).
- The whisper server processes one request at a time; concurrent voice messages queue.

### "Couldn't understand the audio" even though I spoke clearly

whisper returned an empty transcript (no error). The audio may be:
- Too short (sub-second clips rarely transcribe well).
- Silent or near-silent (breathing, background hum with no speech).
- In an unsupported language for the model you're using.
- Encoded in a format whisper doesn't recognise. Try re-encoding as 16 kHz mono WAV.
