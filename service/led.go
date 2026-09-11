package service

import (
	"context"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	blinkRegex = regexp.MustCompile(`(?i)blink\s*:\s*([1-9]|[a-dA-D])`)
)

// ParseBlinkCount extracts a target blink number from a text response.
// 1. Searches anywhere in text for "blink:N" or "blink:A" (case-insensitive).
// 2. Falls back to checking if the trimmed text is a pure single digit (1-9) or option letter (A-D).
func ParseBlinkCount(text string) (int, bool) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return 0, false
	}

	// 1. Check for embedded "blink:N" or "blink:A" anywhere in the response
	matches := blinkRegex.FindStringSubmatch(clean)
	if len(matches) > 1 {
		val := matches[1]
		if count, ok := mapOptionToCount(val); ok {
			return count, true
		}
	}

	// 2. Check if the entire response is a standalone number (1-9)
	if len(clean) == 1 {
		if count, ok := mapOptionToCount(clean); ok {
			return count, true
		}
	}

	// Also check if text is e.g. "Option 3" or "(3)" or "Choice C"
	optionRegex := regexp.MustCompile(`(?i)^(?:option|choice|answer)?\s*[\(\[]?([1-9]|[a-dA-D])[\)\]]?\.?$`)
	if m := optionRegex.FindStringSubmatch(clean); len(m) > 1 {
		if count, ok := mapOptionToCount(m[1]); ok {
			return count, true
		}
	}

	return 0, false
}

func mapOptionToCount(token string) (int, bool) {
	t := strings.TrimSpace(token)
	if len(t) != 1 {
		return 0, false
	}
	ch := t[0]
	if ch >= '1' && ch <= '9' {
		n, err := strconv.Atoi(t)
		if err == nil {
			return n, true
		}
	}
	switch strings.ToUpper(t) {
	case "A":
		return 1, true
	case "B":
		return 2, true
	case "C":
		return 3, true
	case "D":
		return 4, true
	}
	return 0, false
}

// Global LED Manager
type ledManager struct {
	mu             sync.Mutex
	cancel         context.CancelFunc
	lastCount      int
	lastAnswerText string
}

var globalLEDMgr = &ledManager{}

// TriggerLEDIfEnabled parses the text and executes the LED alert and blink sequence if an answer is detected.
func TriggerLEDIfEnabled(text string, choice string) {
	count, found := ParseBlinkCount(text)
	if !found || count <= 0 {
		return
	}

	globalLEDMgr.mu.Lock()
	globalLEDMgr.lastCount = count
	globalLEDMgr.lastAnswerText = text
	// Cancel any currently running sequence
	if globalLEDMgr.cancel != nil {
		globalLEDMgr.cancel()
		globalLEDMgr.cancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalLEDMgr.cancel = cancel
	globalLEDMgr.mu.Unlock()

	log.Printf("[LED] Answer detected: %d blinks requested (Indicator: %s, Stored in RAM)", count, choice)
	go executeLEDSequence(ctx, choice, count)
}

// GetLastAnswerRAM retrieves the last stored answer directly from RAM without touching the clipboard.
func GetLastAnswerRAM() string {
	globalLEDMgr.mu.Lock()
	defer globalLEDMgr.mu.Unlock()
	return globalLEDMgr.lastAnswerText
}

// ReplayLEDIfEnabled replays the most recent LED answer sequence (e.g. on Ctrl + Shift + F).
func ReplayLEDIfEnabled(choice string) {
	globalLEDMgr.mu.Lock()
	count := globalLEDMgr.lastCount
	if count <= 0 {
		globalLEDMgr.mu.Unlock()
		log.Println("[LED] No previous answer sequence to replay.")
		return
	}

	if globalLEDMgr.cancel != nil {
		globalLEDMgr.cancel()
		globalLEDMgr.cancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalLEDMgr.cancel = cancel
	globalLEDMgr.mu.Unlock()

	log.Printf("[LED] Replaying previous answer sequence: %d blinks (Indicator: %s)", count, choice)
	go executeLEDSequence(ctx, choice, count)
}

// executeLEDSequence performs the hardware LED sequence:
// 1. Initial State: Always force OFF.
// 2. Pre-alert signal: Turn ON for 2.5 seconds, then turn OFF.
// 3. Pause for 500ms.
// 4. Blink count phase: Blink ON (400ms) and OFF (400ms) for 'count' times.
// 5. Final State: Always finish in the OFF state.
func executeLEDSequence(ctx context.Context, choice string, count int) {
	vk := resolveVKCode(choice)

	sleepOrCancel := func(d time.Duration) bool {
		select {
		case <-ctx.Done():
			forceHardwareLEDState(vk, false)
			return false
		case <-time.After(d):
			return true
		}
	}

	// 1. Initial State: Always force OFF immediately
	forceHardwareLEDState(vk, false)
	if !sleepOrCancel(200 * time.Millisecond) {
		return
	}

	// 2. Alert / Arrival Phase: Turn ON for 2.5 seconds to indicate answer arrived
	forceHardwareLEDState(vk, true)
	if !sleepOrCancel(2500 * time.Millisecond) {
		return
	}

	// Turn OFF after alert
	forceHardwareLEDState(vk, false)
	if !sleepOrCancel(500 * time.Millisecond) {
		return
	}

	// 3. Blink Phase: Blink 'count' times
	for i := 1; i <= count; i++ {
		// Turn ON
		forceHardwareLEDState(vk, true)
		if !sleepOrCancel(400 * time.Millisecond) {
			return
		}

		// Turn OFF
		forceHardwareLEDState(vk, false)
		if !sleepOrCancel(400 * time.Millisecond) {
			return
		}
	}

	// 4. Final State: Guarantee LED is OFF
	forceHardwareLEDState(vk, false)
	log.Printf("[LED] Sequence complete: %d blinks finished. LED reset to OFF.", count)
}

// PlayLEDStartupSequence runs a fast 5-blink self-test sequence to confirm hardware LED functionality on launch.
func PlayLEDStartupSequence(choice string) {
	vk := resolveVKCode(choice)
	log.Printf("[LED] Running startup self-test (5 quick blinks on %s)...", choice)

	// Ensure initially OFF
	forceHardwareLEDState(vk, false)
	time.Sleep(100 * time.Millisecond)

	for i := 1; i <= 5; i++ {
		forceHardwareLEDState(vk, true)
		time.Sleep(180 * time.Millisecond)
		forceHardwareLEDState(vk, false)
		time.Sleep(180 * time.Millisecond)
	}

	// Guarantee final state is OFF
	forceHardwareLEDState(vk, false)
	log.Printf("[LED] Startup self-test complete. LED indicator ready and set to OFF.")
}

