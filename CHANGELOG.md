# Changelog

All notable changes to WritePilot will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0] — 2026-04-24

### Added

- Initial release: MVP Clipboard Bridge architecture
- Global hotkey support (configurable via `config.json`)
- Multi-LLM provider support:
  - Groq (llama-3.3-70b, free tier with 10k requests/day)
  - OpenAI (gpt-4o-mini)
  - Google Gemini (gemini-2.0-flash)
  - Ollama (local, fully private)
- Two correction modes:
  - **Correct mode**: Returns only the corrected text
  - **Suggest mode**: Returns corrected text + explanation of each change (great for learning)
- Works seamlessly in any Windows app:
  - Microsoft Office (Outlook, Word)
  - Communication apps (Teams, Slack)
  - Text editors (VS Code, Notepad++, Notepad)
  - Web browsers
  - Any application with text input fields
- Windows 10+ toast notifications for completion feedback
- JSON configuration file (no GUI setup needed)
- Cross-app compatibility via simulated Ctrl+C (SendInput API)
- Comprehensive README with setup instructions and examples
- Log output to `writepilot.log` for debugging

### Technical Details

- Written in Go (single binary, ~15 MB)
- Windows-only (leverages Win32 APIs)
- No external dependencies beyond Go standard library and lightweight libraries:
  - `golang.design/x/hotkey` — global hotkey registration
  - `golang.design/x/clipboard` — clipboard read/write
- Runs as background process with no visible window (production build)
- Thread-safe concurrent processing with hotkey debouncing

### Known Limitations

- Windows only (not macOS or Linux in MVP)
- Plan A only: clipboard-based text capture (Plan B: system tray GUI and Plan C: UIAutomation are planned)
- No built-in UI for settings (config via `config.json` file edit)

---

## Planned Features (Roadmap)

### Plan B — System Tray Application (v0.2.0)
- System tray icon with context menu
- Switch language/mode/provider without editing config.json
- Real-time status indicator (processing, idle)
- Settings dialog for easier configuration

### Plan C — Advanced UIAutomation (v0.3.0)
- Direct text read/write from focused window element (no clipboard interference)
- App-aware corrections (e.g., "You're correcting an email in Outlook")
- Correction history with diff view
- Optional auto-replace (automatically write corrected text back to focused field)

### Future (v1.0+)
- Web UI for remote configuration
- Correction history and statistics
- Batch processing for documents
- Custom prompts per app or context
- Integration with external spell-check/grammar engines
- Linux and macOS support

---

## Contributors

- **Initial author**: Created for personal use as a language learning tool for Norwegian learners
- See GitHub for full contributor list

---

## Support

For bug reports, feature requests, and questions, please visit the [Issues](../../issues) page or [Discussions](../../discussions).

