# Contributing to WritePilot

Thanks for your interest in improving WritePilot! We welcome contributions of all kinds — bug reports, feature requests, code improvements, documentation, and translations.

## Code of Conduct

Be respectful and constructive. We aim to be welcoming to people of all backgrounds and skill levels.

---

## How to Contribute

### Report a Bug

1. Check the [Issues](../../issues) page first — your bug may already be reported
2. Open a new issue using the **Bug Report** template
3. Include:
   - Windows version (10, 11, etc.)
   - Go version (if building from source)
   - Steps to reproduce
   - Expected vs. actual behavior
   - Relevant log lines from `writepilot.log`

### Request a Feature

1. Open a new issue using the **Feature Request** template
2. Describe the problem it solves and how it fits WritePilot's goal
3. Examples are always helpful

### Submit Code

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature-name`
3. Make your changes, keeping commits clean and focused
4. Run `go fmt ./...` to format code
5. Run `go vet ./...` to check for issues
6. Commit with clear messages: `git commit -m "Add: feature X"` (use `Add:`, `Fix:`, `Docs:`, `Refactor:` prefixes)
7. Push to your fork: `git push origin feature/your-feature-name`
8. Open a Pull Request with a clear description of what changed and why

### Code Style

- Run `go fmt ./...` before committing (enforced by the language)
- Keep functions focused and well-commented
- Add docstrings to public functions
- Keep lines under 100 characters where reasonable
- Use clear variable names

### Testing Your Changes

1. Build the project:
   ```powershell
   go build -ldflags "-H windowsgui" -o writepilot.exe ./cmd/writepilot
   ```
2. Test manually:
   - Select text in various apps (VS Code, Notepad, Outlook, Teams)
   - Press the hotkey and verify corrections work
   - Check `writepilot.log` for any errors
3. Run static analysis:
   ```powershell
   go vet ./...
   go fmt ./...
   ```

### Adding a New LLM Provider

See `docs/ADDING_LLM_PROVIDERS.md` for detailed instructions.

---

## Development Setup

### Prerequisites

- Go 1.21 or later
- Windows 10 or later
- Git

### Clone and Build

```powershell
git clone https://github.com/YOUR_USERNAME/writepilot.git
cd writepilot
go mod download
go build -ldflags "-H windowsgui" -o writepilot.exe ./cmd/writepilot
```

### Development Build (with console window for debugging)

```powershell
go build -o writepilot.exe ./cmd/writepilot
.\writepilot.exe
# Watch writepilot.log in another terminal:
Get-Content writepilot.log -Tail 5 -Wait
```

---

## Project Structure

```
cmd/writepilot/          — Entry point, hotkey loop
internal/config/         — JSON configuration loading
internal/capture/        — Ctrl+C simulation, clipboard handling
internal/llm/            — LLM client (OpenAI-compatible)
internal/notify/         — Windows toast notifications
README.md                — User documentation
CONTRIBUTING.md          — This file
CHANGELOG.md             — Version history
```

---

## Commit Message Conventions

Use these prefixes for clarity:

- `Add:` — new feature
- `Fix:` — bug fix
- `Docs:` — documentation only
- `Refactor:` — code reorganization without behavior change
- `Test:` — testing additions/fixes (if added later)
- `Chore:` — build, CI, dependency updates

Example:
```
Fix: handle Notepad clipboard delays with extended polling

Previously, Notepad's slower clipboard handling caused WritePilot
to timeout and report "no text selected". Extended the polling
window to 2s and added immediate clipboard check.
```

---

## Questions?

- Check existing [Issues](../../issues) and [Discussions](../../discussions)
- Open a Discussion if you have a question
- Read the README and existing code comments

Thank you for contributing! 🙏
