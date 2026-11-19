# WritePilot

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/dl/)
[![Windows 10+](https://img.shields.io/badge/Windows-10+-0078d4.svg)](https://www.microsoft.com/windows/)

A lightweight Windows background tool that corrects grammar and spelling in **any** text field — Outlook, Word, Teams, Slack, Notepad, browsers — without leaving the app. Built for non-native speakers who write daily: in **suggest** mode it also explains every change, so you learn from your mistakes.

## How it works

1. Select text in any app and press the hotkey (default `Ctrl+Shift+Space`).
2. WritePilot copies the selection, sends it to your chosen LLM, and puts the corrected text on the clipboard.
3. A toast notification appears. Paste with `Ctrl+V`, or let `auto_paste` do it for you.

## Build

Requires Windows 10+ and Go 1.25+.

```powershell
# Debug build (console window visible, shows logs):
go build -o writepilot.exe ./cmd/writepilot

# Production build (runs silently in the background):
go build -ldflags "-H windowsgui" -o writepilot.exe ./cmd/writepilot
```

Create your config from the template, then put `writepilot.exe` and `config.json` in the same folder and run it. To start it with Windows, place a shortcut to it in `shell:startup`.

```powershell
Copy-Item config.example.json config.json
```

## Configuration

`config.json` lives next to `writepilot.exe`. It is gitignored, so your API key stays local; only `config.example.json` is committed:

```json
{
  "language":        "",
  "mode":            "correct",
  "hotkey":          "Ctrl+Shift+Space",
  "provider":        "groq",
  "api_key":         "YOUR_API_KEY_HERE",
  "api_base_url":    "",
  "model":           "",
  "fallbacks": [
    { "provider": "gemini", "api_key": "YOUR_GEMINI_KEY", "model": "" }
  ],
  "auto_paste":      false,
  "timeout_seconds": 30,
  "attempt_timeout_seconds": 10
}
```

| Field | Default | Description |
|---|---|---|
| `language` | *(auto-detect)* | Target language, e.g. `English`, `Norwegian`. Empty lets the LLM detect it. |
| `mode` | `correct` | `correct` returns only the fixed text. `suggest` also lists each correction with an explanation. |
| `hotkey` | `Ctrl+Shift+Space` | Modifiers `Ctrl`, `Shift`, `Alt`, `Win` plus a key: `Space`, `Return`, `F1`–`F12`, `A`–`Z`, `0`–`9`. |
| `provider` | `groq` | `groq`, `openai`, `gemini` or `ollama`. |
| `api_key` | — | API key for the provider. Leave empty for `ollama`. |
| `api_base_url` | *(per provider)* | Override the provider's endpoint. |
| `model` | *(per provider)* | Override the model (defaults below). |
| `fallbacks` | `[]` | Extra providers tried in order when the primary fails. Each entry takes `provider`, `api_key`, `api_base_url` and `model`, with the same defaults. |
| `auto_paste` | `false` | Paste the corrected text automatically with `Ctrl+V`. |
| `timeout_seconds` | `30` | Total time for one correction, across all retries and fallbacks. |
| `attempt_timeout_seconds` | `10` | Time for a single request before WritePilot gives up on it. Raise it for slow local models. |

### Retries and fallbacks

When a provider is overloaded (503), rate-limited (429), has a server error or times out, WritePilot retries it once after a short pause. Then it moves to the next entry in `fallbacks`. A timeout skips the retry when a fallback exists, and other errors (bad key, unknown model) go straight to the next provider. The toast tells you when a fallback answered and why; every attempt, with the provider's own error message, is logged to `writepilot.log`.

### Providers

| Provider | Default model | Get a key |
|---|---|---|
| `groq` | `qwen/qwen3.8-27b` | [console.groq.com](https://console.groq.com) |
| `gemini` | `gemini-3.1-flash-lite` | [aistudio.google.com](https://aistudio.google.com) |
| `openai` | `gpt-4o-mini` | [platform.openai.com](https://platform.openai.com) |
| `ollama` | `llama3.1` | Local, no key — install from [ollama.com](https://ollama.com), then `ollama pull llama3.1` |

> ⚠️ **Privacy:** with cloud providers your text is sent to external servers. Don't send passwords, personal data or confidential content — use Ollama for sensitive text, it never leaves your machine.

### Suggest mode example

Only the corrected text is pasted (or copied); the corrections appear in the notification:

```
--- Corrections ---
• Original: "Jeg gik hjem" → Corrected: "Jeg gikk hjem" — Past tense of "gå" is "gikk", not "gik"
• Original: "for at lære" → Corrected: "for å lære" — Use the infinitive "å lære" after "for"
```

## Troubleshooting

Logs are written to `writepilot.log` next to the executable (`Get-Content .\writepilot.log -Tail 20`).

| Symptom | Fix |
|---|---|
| Nothing happens on the hotkey | Another app may own it — change `hotkey`. |
| "All providers failed: … overloaded (503)" | The provider is out of capacity, not your quota. Add a fallback provider. |
| "All providers failed: … HTTP 401" | Invalid API key for that provider; the log has the provider's message. |
| `config.json not found` | Put `config.json` in the same folder as `writepilot.exe`. |
| No toast notification | Check Focus Assist / Do Not Disturb. |

## License

MIT
