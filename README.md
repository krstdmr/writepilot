# WritePilot

**For non-native language learners who write daily but want to improve without interrupting their workflow.**

## Why WritePilot?

Imagine you're at work in Norway, writing in Norwegian. A customer needs an urgent reply, and you've typed a quick email — but you're unsure if your grammar is perfect. Do you risk sending it with mistakes, or spend some time copying it to ChatGPT, editing it, and pasting back?

WritePilot eliminates that friction. Press a hotkey. Done. Your text is corrected in seconds, right where you're typing — in Outlook, Teams, Slack, Word, or any app. No context switching. No tab juggling.

**But here's what makes it different:** In "suggest" mode, WritePilot doesn't just fix your text — it *explains* every change. You see exactly where you went wrong and why. Over time, you stop making those mistakes. You learn.

This tool is designed for language learners like you who want to:
- ✅ Write faster without worrying about every typo
- ✅ Keep working in your chosen app (no copy-paste to web tools)
- ✅ Get instant corrections with explanations, so mistakes become lessons
- ✅ Improve your Norwegian (or any language) through daily practice, not formal study
- ✅ Choose any LLM provider — cloud-based (Groq, OpenAI, Gemini) or **100% private with local Ollama**

Perfect for emails, chat messages, documentation, or any urgent writing task where correctness matters.

---

## How it works

A lightweight Windows background tool that corrects grammar and spelling in **any** text field — Outlook, Word, Teams, Slack, Notepad, browsers — without leaving the app you are working in.

Press a hotkey while text is selected, wait a moment, then paste the corrected result with `Ctrl+V`. Optionally, enable **suggest** mode to also get an explanation of every change, so you learn from your mistakes.

**Flexibility:** Choose your LLM provider in `config.json` — swap between cloud APIs (Groq, OpenAI, Google Gemini) or run **locally with Ollama** for complete privacy with zero API costs.

```
Select text in any app
        │
        ▼
Press hotkey (default: Ctrl+Shift+Space)
        │
        ▼
WritePilot simulates Ctrl+C → reads the clipboard
        │
        ▼
Sends the text to your chosen LLM with a correction prompt
        │
        ▼
Writes the corrected text back to the clipboard
        │
        ▼
Windows toast notification: "Corrected text copied. Paste with Ctrl+V."
        │
        ▼
Press Ctrl+V in your app → done
```

---

## Requirements

| Requirement | Details |
|---|---|
| OS | Windows 10 or later |
| Go | 1.21 or later (for building from source) |
| LLM provider | Groq, OpenAI, Google Gemini, or local Ollama |

---

## Installation

### Option A — Build from source

```powershell
git clone <repo-url>
cd write-pilot-plugin

go mod tidy

# Development build (console window visible, useful for watching logs):
go build -o writepilot.exe ./cmd/writepilot

# Production build (no console window, runs silently in the background):
go build -ldflags "-H windowsgui" -o writepilot.exe ./cmd/writepilot
```

Copy `writepilot.exe` and `config.json` to any folder, e.g. `C:\Tools\WritePilot\`.

### Option B — Pre-built binary

Download `writepilot.exe` from the Releases page, place it in a folder alongside `config.json`.

---

## Configuration

Edit `config.json` in the same directory as `writepilot.exe`:

```json
{
  "language":        "Norwegian",
  "mode":            "correct",
  "hotkey":          "Ctrl+Shift+Space",
  "provider":        "groq",
  "api_key":         "YOUR_API_KEY_HERE",
  "api_base_url":    "",
  "model":           "",
  "timeout_seconds": 30
}
```

### Fields

| Field | Default | Description |
|---|---|---|
| `language` | `Norwegian` | The language to correct toward. Any language name works, e.g. `English`, `German`, `Spanish`. |
| `mode` | `correct` | `correct` — return only the fixed text. `suggest` — return fixed text **plus** a list of corrections with explanations (great for learning). |
| `hotkey` | `Ctrl+Shift+Space` | The global keyboard shortcut that triggers WritePilot. See [Hotkey format](#hotkey-format). |
| `provider` | `groq` | LLM backend: `groq`, `openai`, `gemini`, `ollama`. |
| `api_key` | *(required)* | Your API key for the chosen provider. Leave empty for `ollama`. |
| `api_base_url` | *(auto)* | Override the provider's base URL. Leave empty to use the built-in default. |
| `model` | *(auto)* | Override the model name. Leave empty to use the provider's recommended default. |
| `timeout_seconds` | `30` | How long to wait for an LLM response before giving up. |

### Hotkey format

```
<modifier>+<modifier>+<key>
```

**Modifiers:** `Ctrl`, `Shift`, `Alt`, `Win`

**Keys:** `Space`, `Return`, `F1`–`F12`, `A`–`Z`, `0`–`9`

Examples:
- `Ctrl+Shift+Space` *(default)*
- `Ctrl+Alt+C`
- `Alt+F9`

> **Tip:** Avoid hotkeys that are already used by your applications (e.g. `Ctrl+C`, `Ctrl+V`).

---

## LLM providers

### Groq *(recommended — fast, generous free tier)*

1. Sign up at [console.groq.com](https://console.groq.com)
2. Create an API key under **API Keys**
3. Set `"provider": "groq"` and paste the key into `"api_key"`
4. Default model: `llama-3.3-70b-versatile`

Free tier: ~10,000 requests/day, no credit card required.

---

### Google Gemini *(excellent Norwegian quality)*

1. Sign up at [aistudio.google.com](https://aistudio.google.com)
2. Click **Get API key**
3. Set in `config.json`:
   ```json
   "provider": "gemini",
   "api_key": "YOUR_GEMINI_KEY"
   ```
4. Default model: `gemini-2.0-flash`

Free tier: 1,500 requests/day, no credit card required.

---

### OpenAI

1. Sign up at [platform.openai.com](https://platform.openai.com)
2. Go to **API Keys** and create a key
3. Set in `config.json`:
   ```json
   "provider": "openai",
   "api_key": "YOUR_OPENAI_KEY"
   ```
4. Default model: `gpt-4o-mini`

New accounts receive $5 of free credits.

---

### Ollama *(free, fully local, private)*

1. Install Ollama from [ollama.com](https://ollama.com)
2. Pull a model: `ollama pull llama3.1`
3. Make sure Ollama is running (`ollama serve`)
4. Set in `config.json`:
   ```json
   "provider": "ollama",
   "api_key":  "",
   "model":    "llama3.1"
   ```

No internet required. Text never leaves your machine.

---

## Running WritePilot

**During development** (console visible):
```powershell
.\writepilot.exe
```

**Daily use** (no console window, silent background process):
```powershell
# Build without console first:
go build -ldflags "-H windowsgui" -o writepilot.exe ./cmd/writepilot

# Then just double-click writepilot.exe, or add to startup
```

### Run at Windows startup (optional)

1. Press `Win+R`, type `shell:startup`, press Enter
2. Create a shortcut to `writepilot.exe` in that folder

---

## Usage

### Basic workflow

1. In any application, **select the text** you want to correct
2. Press the hotkey (default `Ctrl+Shift+Space`)
3. A moment later, a Windows notification appears:
   > *WritePilot — Done: Corrected text copied. Paste with Ctrl+V.*
4. Press `Ctrl+V` to paste the corrected text

### Real-world examples

**Urgent customer email (Outlook)**
- You've typed a quick reply to a customer in Outlook and need to send it in 2 minutes
- Select your message, press `Ctrl+Shift+Space`, paste — your email is now grammatically correct
- No switching tabs, no leaving Outlook, no delays

**Quick Slack message to a colleague**
- You want to ask something in Norwegian but worry about typos slowing down communication
- Write in the message box, press the hotkey, paste — your message is polished before hitting Send

**Documentation or support ticket**
- You're filling out an urgent support form or writing internal documentation in Norwegian
- Select a paragraph, press the hotkey, paste — you look professional, grammar is perfect

**Suggest mode for learning**
- Enable `"mode": "suggest"` in config.json
- Press the hotkey — instead of just the corrected text, you also get:
  ```
  --- Corrections ---
  • Original: "Jeg gik hjem" → Corrected: "Jeg gikk hjem" — Past tense of "gå" is "gikk", not "gik"
  • Original: "for at lære" → Corrected: "for å lære" — Use the infinitive "å lære" after "for"
  ```
- Review the corrections, paste the corrected text, and **remember the rule next time**

---

## Logs

WritePilot writes logs to `writepilot.log` in the same directory as the executable. If something goes wrong (LLM error, hotkey conflict, etc.), check that file first.

```powershell
Get-Content .\writepilot.log -Tail 20
```

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| Nothing happens when pressing the hotkey | Check `writepilot.log`. Another app may own that hotkey — change `hotkey` in `config.json`. |
| "LLM request failed" toast | Check `writepilot.log` for the exact error. Usually an invalid API key or network issue. |
| Paste produces garbled text | The app may have blocked `Ctrl+C`. Try selecting the text more carefully, or use a plain-text editor as an intermediary. |
| `config.json not found` on startup | Make sure `config.json` is in the same folder as `writepilot.exe`. |
| No toast notification | Toast notifications require Windows 10+ and must not be suppressed by Focus Assist / Do Not Disturb settings. |

---

## Project structure

```
write-pilot-plugin/
├── cmd/
│   └── writepilot/
│       └── main.go          ← entry point, hotkey loop, pipeline
├── internal/
│   ├── config/
│   │   └── config.go        ← JSON config loading and defaults
│   ├── capture/
│   │   └── capture.go       ← Ctrl+C simulation + clipboard read (Windows)
│   ├── llm/
│   │   └── client.go        ← OpenAI-compatible HTTP client
│   └── notify/
│       └── notify.go        ← Windows toast notification via PowerShell
├── config.json              ← user configuration
├── go.mod
└── README.md
```

---

## License

MIT
