//go:build !windows

package service

func resolveVKCode(choice string) uintptr {
	return 0
}

func isHardwareLEDOn(vk uintptr) bool {
	return false
}

func toggleHardwareLED(vk uintptr) {}

func forceHardwareLEDState(vk uintptr, targetOn bool) {}
