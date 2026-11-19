//go:build windows

package capture

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.design/x/clipboard"
)

var (
	user32                     = syscall.MustLoadDLL("user32.dll")
	sendInput                  = user32.MustFindProc("SendInput")
	getForegroundWindow        = user32.MustFindProc("GetForegroundWindow")
	getClipboardSequenceNumber = user32.MustFindProc("GetClipboardSequenceNumber")
)

// ForegroundWindow returns a handle to the window that currently has focus.
func ForegroundWindow() uintptr {
	h, _, _ := getForegroundWindow.Call()
	return h
}

// clipboardSequence returns the clipboard's change counter, which Windows
// increments every time any program writes to the clipboard (0 if unavailable).
func clipboardSequence() uintptr {
	n, _, _ := getClipboardSequenceNumber.Call()
	return n
}

const (
	inputKeyboard  = uint32(1)
	keyeventfKeyup = uint32(0x0002)

	vkShift   = uint16(0x10) // VK_SHIFT
	vkControl = uint16(0x11) // VK_CONTROL
	vkAlt     = uint16(0x12) // VK_MENU (Alt)
	vkWin     = uint16(0x5B) // VK_LWIN
	vkC       = uint16(0x43) // VK_C
	vkV       = uint16(0x56) // VK_V
)

// keyboardINPUT mirrors the Windows INPUT structure for keyboard events.
//
// Layout on 64-bit Windows (sizeof(INPUT) == 40):
//
//	Offset  0: DWORD  type       (4 bytes)
//	Offset  4: [4]byte padding   (4 bytes — aligns the 8-byte pointer in the union)
//	Offset  8: WORD   wVk        (2 bytes)
//	Offset 10: WORD   wScan      (2 bytes)
//	Offset 12: DWORD  dwFlags    (4 bytes)
//	Offset 16: DWORD  time       (4 bytes)
//	Offset 20: [4]byte padding   (4 bytes — aligns ULONG_PTR dwExtraInfo to 8)
//	Offset 24: ULONG_PTR extra   (8 bytes)
//	Offset 32: [8]byte padding   (8 bytes — union padded to sizeof(MOUSEINPUT)=32)
//	Total: 40 bytes
type keyboardINPUT struct {
	inputType uint32
	_         [4]byte
	wVk       uint16
	wScan     uint16
	dwFlags   uint32
	timestamp uint32
	_         [4]byte
	dwExtra   uintptr
	_         [8]byte
}

// pressKey sends a single key event via the modern SendInput API.
func pressKey(vk uint16, flags uint32) {
	inp := keyboardINPUT{
		inputType: inputKeyboard,
		wVk:       vk,
		dwFlags:   flags,
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

// releaseModifiers sends key-up events for any modifier keys that may still be
// physically held by the user at the time the hotkey fires. Without this,
// a hotkey like Ctrl+Shift+Space would leave Shift held, so the injected
// Ctrl+C would arrive at the target app as Ctrl+Shift+C — which does nothing.
func releaseModifiers() {
	for _, vk := range []uint16{vkControl, vkShift, vkAlt, vkWin} {
		pressKey(vk, keyeventfKeyup)
	}
	// Let the OS deliver the key-up events before we inject Ctrl+C.
	time.Sleep(50 * time.Millisecond)
}

// simulateCtrlC releases any held modifier keys from the triggering hotkey,
// then sends a clean Ctrl+C to the currently active window using SendInput.
// This works reliably with all Windows applications including Electron apps
// (VS Code, Teams, Slack) and Win32 apps (Outlook, Word, Notepad).
func simulateCtrlC() {
	releaseModifiers()                  // drop Ctrl/Shift/Alt held from hotkey press
	pressKey(vkControl, 0)              // Ctrl down
	pressKey(vkC, 0)                    // C down
	time.Sleep(50 * time.Millisecond)   // hold briefly so the app sees the chord
	pressKey(vkC, keyeventfKeyup)       // C up
	pressKey(vkControl, keyeventfKeyup) // Ctrl up
	// No fixed wait here: GetSelectedText polls the clipboard, which covers
	// slow apps (e.g. Notepad) without delaying fast ones.
}

// GetSelectedText simulates Ctrl+C to copy whatever text is selected in the
// foreground application and returns it. It detects the copy through the
// clipboard's change counter, so selecting the same text that is already on
// the clipboard still works. Returns an error if nothing was copied within
// 2 seconds or the copied content has no text.
func GetSelectedText() (string, error) {
	seqBefore := clipboardSequence()
	previous := string(clipboard.Read(clipboard.FmtText)) // only used if the counter is unavailable

	simulateCtrlC()

	// Poll every 20ms for up to 2 seconds.
	deadline := time.Now().Add(2 * time.Second)
	changed := false
	for {
		var current string
		if seqBefore != 0 {
			if clipboardSequence() != seqBefore {
				changed = true
				// The app may still be writing; an empty read means "not yet".
				current = string(clipboard.Read(clipboard.FmtText))
			}
		} else if c := string(clipboard.Read(clipboard.FmtText)); c != previous {
			changed, current = true, c
		}
		if current != "" {
			return current, nil
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if changed {
		return "", fmt.Errorf("the copied selection contains no text")
	}
	return "", fmt.Errorf("no text selected — select some text before pressing the hotkey")
}

// PasteText simulates Ctrl+V to paste the clipboard content into the active
// window. This is used for auto-paste functionality to automatically insert
// the corrected text without requiring manual paste.
func PasteText() {
	releaseModifiers()                  // drop any modifiers held from previous operations
	pressKey(vkControl, 0)              // Ctrl down
	pressKey(vkV, 0)                    // V down
	time.Sleep(50 * time.Millisecond)   // hold briefly so the app sees the chord
	pressKey(vkV, keyeventfKeyup)       // V up
	pressKey(vkControl, keyeventfKeyup) // Ctrl up
	time.Sleep(100 * time.Millisecond)  // wait for the paste operation to complete
}
