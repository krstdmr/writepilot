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
	user32    = syscall.MustLoadDLL("user32.dll")
	sendInput = user32.MustFindProc("SendInput")
)

const (
	inputKeyboard  = uint32(1)
	keyeventfKeyup = uint32(0x0002)

	vkShift   = uint16(0x10) // VK_SHIFT
	vkControl = uint16(0x11) // VK_CONTROL
	vkAlt     = uint16(0x12) // VK_MENU (Alt)
	vkWin     = uint16(0x5B) // VK_LWIN
	vkC       = uint16(0x43) // VK_C
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
	time.Sleep(400 * time.Millisecond)  // extended wait for older Win32 apps like Notepad
}

// GetSelectedText saves the current clipboard content, simulates Ctrl+C to
// copy whatever text is selected in the foreground application, then returns
// the newly copied text. Returns an error if nothing was selected or the
// clipboard did not change within 2 seconds.
func GetSelectedText() (string, error) {
	// Snapshot what is currently in the clipboard.
	previous := string(clipboard.Read(clipboard.FmtText))

	// Send Ctrl+C to the active window.
	simulateCtrlC()

	// Check clipboard immediately (without sleep) for fast apps,
	// then poll with 50ms intervals for up to 2 seconds total.
	deadline := time.Now().Add(2 * time.Second)
	for {
		current := string(clipboard.Read(clipboard.FmtText))
		if current != previous && current != "" {
			return current, nil
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return "", fmt.Errorf("no text selected — select some text before pressing the hotkey")
}
