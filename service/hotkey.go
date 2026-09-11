package service

type HotkeyHandler struct {
	onScreenshot    func()
	onFetchText     func()
	onSendText      func()
	onToggleOverlay func()
	enableXZ        bool
	threadID        uint32
	stopChan        chan struct{}
}

func NewHotkeyHandler(onScreenshot func(), onFetchText func(), onSendText func(), onToggleOverlay func()) *HotkeyHandler {
	return &HotkeyHandler{
		onScreenshot:    onScreenshot,
		onFetchText:     onFetchText,
		onSendText:      onSendText,
		onToggleOverlay: onToggleOverlay,
		enableXZ:        false,
		stopChan:        make(chan struct{}),
	}
}

func (h *HotkeyHandler) SetEnableXZ(enabled bool) {
	h.enableXZ = enabled
}
