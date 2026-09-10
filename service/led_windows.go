//go:build windows

package service

import (
	"strings"
	"syscall"
	"time"
)

const (
	VK_CAPITAL            = 0x14 // Caps Lock Virtual-Key code
	VK_NUMLOCK            = 0x90 // Num Lock Virtual-Key code
	KEYEVENTF_EXTENDEDKEY = 0x0001
	KEYEVENTF_KEYUP       = 0x0002
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procGetKeyState   = user32.NewProc("GetKeyState")
	procKeybdEvent    = user32.NewProc("keybd_event")
	procMapVirtualKey = user32.NewProc("MapVirtualKeyW")
)

func resolveVKCode(choice string) uintptr {
	c := strings.ToLower(strings.TrimSpace(choice))
	if c == "num_lock" || c == "numlock" || c == "num" {
		return uintptr(VK_NUMLOCK)
	}
	return uintptr(VK_CAPITAL)
}

// isHardwareLEDOn returns true if the specified key toggle state is ON (bit 0 is set)
func isHardwareLEDOn(vk uintptr) bool {
	ret, _, _ := procGetKeyState.Call(vk)
	return (ret & 1) != 0
}

// toggleHardwareLED simulates key press and release to toggle the hardware LED
func toggleHardwareLED(vk uintptr) {
	scanCode, _, _ := procMapVirtualKey.Call(vk, 0)
	if scanCode == 0 {
		scanCode = 0x45
	}

	// Key Down
	procKeybdEvent.Call(
		vk,
		scanCode,
		uintptr(KEYEVENTF_EXTENDEDKEY),
		0,
	)
	// Key Up
	procKeybdEvent.Call(
		vk,
		scanCode,
		uintptr(KEYEVENTF_EXTENDEDKEY|KEYEVENTF_KEYUP),
		0,
	)
}

// forceHardwareLEDState ensures the LED reaches the target state (ON or OFF)
func forceHardwareLEDState(vk uintptr, targetOn bool) {
	if isHardwareLEDOn(vk) != targetOn {
		toggleHardwareLED(vk)
		time.Sleep(50 * time.Millisecond)
		// Double-check in case system was slow to register
		if isHardwareLEDOn(vk) != targetOn {
			toggleHardwareLED(vk)
			time.Sleep(50 * time.Millisecond)
		}
	}
}
