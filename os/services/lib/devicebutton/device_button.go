package devicebutton

import (
	"context"
	"log/slog"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

const (
	gpioResetButton = 26
	pollInterval    = 200 * time.Millisecond
	// pressMaxDuration is the max hold time that still counts as a short press (tap).
	pressMaxDuration = 500 * time.Millisecond
)

// DeviceButton watches GPIO 26 (reset button). It provides two callbacks:
//   - onPress: called on short tap (release within pressMaxDuration).
//   - onPressAndHold(duration, released): called each poll tick while held
//     (released=false) and once on release (released=true). Not called for short taps.
//
// Button is active-low with internal pull-up (press = pin to GND).
type DeviceButton struct {
	chip *gpiocdev.Chip
	line *gpiocdev.Line

	started bool
	cancel  context.CancelFunc
}

// ProvideDeviceButton creates an uninitialized DeviceButton. Call Init to open GPIO.
func ProvideDeviceButton() *DeviceButton {
	return &DeviceButton{}
}

// Init opens gpiochip0 and requests the GPIO line with pull-up.
// Returns error when GPIO is unavailable (e.g. dev machine).
func (s *DeviceButton) Init() error {
	chip, err := gpiocdev.NewChip("gpiochip0")
	if err != nil {
		return err
	}
	line, err := chip.RequestLine(gpioResetButton, gpiocdev.AsInput, gpiocdev.WithPullUp)
	if err != nil {
		_ = chip.Close()
		return err
	}
	s.chip = chip
	s.line = line
	return nil
}

// Start begins watching the button in a goroutine. Init must be called first.
// Call Close when done.
func (s *DeviceButton) Start(ctx context.Context, onPress func(), onPressAndHold func(duration time.Duration, released bool)) {
	if s.started || s.line == nil {
		return
	}
	s.started = true
	ctx, s.cancel = context.WithCancel(ctx)

	go s.run(ctx, onPress, onPressAndHold)
}

func (s *DeviceButton) run(ctx context.Context, onPress func(), onPressAndHold func(duration time.Duration, released bool)) {
	var holdStart time.Time
	holdReported := false
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			val, err := s.line.Value()
			if err != nil {
				slog.Error("read GPIO failed", "component", "devicebutton", "gpio", gpioResetButton, "error", err)
				holdStart = time.Time{}
				holdReported = false
				continue
			}
			// Active low: 0 = pressed, 1 = released
			if val == 0 {
				if holdStart.IsZero() {
					holdStart = time.Now()
				}
				elapsed := time.Since(holdStart)
				if elapsed >= pressMaxDuration {
					holdReported = true
					if onPressAndHold != nil {
						onPressAndHold(elapsed, false)
					}
				}
			} else {
				if !holdStart.IsZero() {
					elapsed := time.Since(holdStart)
					if holdReported {
						if onPressAndHold != nil {
							onPressAndHold(elapsed, true)
						}
					} else if onPress != nil {
						onPress()
					}
				}
				holdStart = time.Time{}
				holdReported = false
			}
		}
	}
}

// Close cancels the run goroutine, waits for it to exit,
// then releases the GPIO line and chip.
func (s *DeviceButton) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.started = false
	if s.line != nil {
		_ = s.line.Close()
		s.line = nil
	}
	if s.chip != nil {
		return s.chip.Close()
	}
	return nil
}
