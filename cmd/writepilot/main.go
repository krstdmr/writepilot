package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"golang.design/x/clipboard"
	"golang.design/x/hotkey"

	"writepilot/internal/capture"
	"writepilot/internal/config"
	"writepilot/internal/llm"
	"writepilot/internal/notify"
)

// processing prevents concurrent pipeline runs when the user presses the
// hotkey multiple times before the LLM has responded.
var processing atomic.Bool

func main() {
	setupLogging()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[WritePilot] config error: %v", err)
	}

	if err := clipboard.Init(); err != nil {
		log.Fatalf("[WritePilot] clipboard init failed: %v", err)
	}

	mods, key, err := parseHotkey(cfg.Hotkey)
	if err != nil {
		log.Fatalf("[WritePilot] invalid hotkey %q: %v", cfg.Hotkey, err)
	}

	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Fatalf("[WritePilot] failed to register hotkey %q: %v\n"+
			"  Another application may already use this hotkey. Change 'hotkey' in config.json.", cfg.Hotkey, err)
	}
	defer hk.Unregister()

	log.Printf("[WritePilot] running — hotkey: %s | language: %s | mode: %s | provider: %s (%s)",
		cfg.Hotkey, cfg.Language, cfg.Mode, cfg.Provider, cfg.Model)

	// Block on hotkey events; each press spawns a goroutine for the pipeline.
	for range hk.Keydown() {
		if !processing.CompareAndSwap(false, true) {
			log.Println("[WritePilot] already processing — ignoring hotkey press")
			continue
		}
		go func(c *config.Config) {
			defer processing.Store(false)
			runPipeline(c)
		}(cfg)
	}
}

// runPipeline is the core workflow:
//  1. Capture selected text from the active window via simulated Ctrl+C
//  2. Send it to the configured LLM for correction
//  3. Write the result to the clipboard
//  4. Fire a Windows toast so the user knows to paste
func runPipeline(cfg *config.Config) {
	log.Println("[WritePilot] hotkey pressed — capturing selected text…")

	text, err := capture.GetSelectedText()
	if err != nil {
		log.Printf("[WritePilot] capture: %v", err)
		return // silent — user probably didn't select anything
	}

	log.Printf("[WritePilot] captured %d chars — calling %s…", len(text), cfg.Provider)

	result, err := llm.Correct(cfg, text)
	if err != nil {
		log.Printf("[WritePilot] LLM error: %v", err)
		// Notify the user so they know something went wrong.
		_ = notify.Toast("WritePilot — Error", "LLM request failed. Check writepilot.log for details.")
		return
	}

	clipboard.Write(clipboard.FmtText, []byte(result))
	log.Printf("[WritePilot] done — corrected text copied to clipboard (%d chars)", len(result))

	// Auto-paste if enabled
	if cfg.AutoPaste {
		// Give the clipboard write a moment to settle
		time.Sleep(100 * time.Millisecond)
		capture.PasteText()
		log.Println("[WritePilot] auto-pasted corrected text")
		_ = notify.Toast("WritePilot — Done", "Corrected text pasted!")
	} else {
		_ = notify.Toast("WritePilot — Done", "Corrected text copied. Paste with Ctrl+V.")
	}
}

// setupLogging configures the logger to write to writepilot.log next to the
// executable, falling back to stderr if the file cannot be created.
func setupLogging() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exe), "writepilot.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
}

// parseHotkey parses a string such as "Ctrl+Shift+Space" into the modifier
// list and key value expected by golang.design/x/hotkey.
func parseHotkey(hkStr string) ([]hotkey.Modifier, hotkey.Key, error) {
	parts := strings.Split(hkStr, "+")
	if len(parts) < 2 {
		return nil, 0, fmt.Errorf("use format like Ctrl+Shift+Space (got %q)", hkStr)
	}

	keyStr := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
	modStrs := parts[:len(parts)-1]

	var mods []hotkey.Modifier
	for _, m := range modStrs {
		switch strings.ToLower(strings.TrimSpace(m)) {
		case "ctrl":
			mods = append(mods, hotkey.ModCtrl)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		case "alt":
			mods = append(mods, hotkey.ModAlt)
		case "win":
			mods = append(mods, hotkey.ModWin)
		default:
			return nil, 0, fmt.Errorf("unknown modifier %q (supported: Ctrl, Shift, Alt, Win)", m)
		}
	}

	key, err := parseKey(keyStr)
	if err != nil {
		return nil, 0, err
	}

	return mods, key, nil
}

// parseKey maps a key name string to its Windows Virtual Key code.
// golang.design/x/hotkey uses Windows VK codes as its Key type on Windows.
func parseKey(s string) (hotkey.Key, error) {
	// Named keys
	switch s {
	case "space":
		return hotkey.Key(0x20), nil
	case "return", "enter":
		return hotkey.Key(0x0D), nil
	case "f1":
		return hotkey.Key(0x70), nil
	case "f2":
		return hotkey.Key(0x71), nil
	case "f3":
		return hotkey.Key(0x72), nil
	case "f4":
		return hotkey.Key(0x73), nil
	case "f5":
		return hotkey.Key(0x74), nil
	case "f6":
		return hotkey.Key(0x75), nil
	case "f7":
		return hotkey.Key(0x76), nil
	case "f8":
		return hotkey.Key(0x77), nil
	case "f9":
		return hotkey.Key(0x78), nil
	case "f10":
		return hotkey.Key(0x79), nil
	case "f11":
		return hotkey.Key(0x7A), nil
	case "f12":
		return hotkey.Key(0x7B), nil
	}

	// Single letter (A–Z) or digit (0–9)
	if len(s) == 1 {
		r := unicode.ToUpper(rune(s[0]))
		if r >= 'A' && r <= 'Z' {
			return hotkey.Key(r), nil // VK_A = 0x41 = 'A', ..., VK_Z = 0x5A = 'Z'
		}
		if r >= '0' && r <= '9' {
			return hotkey.Key(r), nil // VK_0 = 0x30 = '0', ..., VK_9 = 0x39 = '9'
		}
	}

	return 0, fmt.Errorf("unsupported key %q — supported: Space, Return, F1–F12, A–Z, 0–9", s)
}
