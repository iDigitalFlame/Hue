package hue

import (
	"time"
)

// ErrNoColor is an error returned when attempting to set the color on a Light when the Light does not support
// colors, meaning it is only has white support.
var ErrNoColor = &errval{s: `light does not support color`}

// Light represents a controllable Hue Light. This can be used to control and set the Light State.
type Light struct {
	gamut *gamut
	Control
}

// Hue returns the hue color of the Light, if set.
func (l Light) Hue() uint16 {
	return l.state.Hue
}

// IsColor returns true if the Light supports colors.
func (l Light) IsColor() bool {
	return bool(l.state.Color)
}

// Effect returns a representation of the color effect that can be set.
func (l Light) Effect() Effect {
	return l.state.Effect
}

// Brightness returns the brightness level of the Light.
func (l Light) Brightness() uint8 {
	return l.state.Brightness
}

// Saturation returns the color saturation level of the Light.
func (l Light) Saturation() uint8 {
	return l.state.Saturation
}

// Temperature returns the color temperature level of the Light.
func (l Light) Temperature() uint16 {
	return l.state.Temperature
}

// XY returns the set color of the Light on the CIE 1931 XY axis.
func (l Light) XY() (float32, float32) {
	return l.state.XY[0], l.state.XY[1]
}

// SetHue will set the color hue of the Light to the specified value. This function returns any errors during setting
// the state. This function immediately returns if the 'Manual' attribute is "true" and will change the state once
// the 'Update*'function is called. Returns ErrNoColor if the Light does not support color.
func (l *Light) SetHue(h uint16) error {
	if !l.state.Color {
		return ErrNoColor
	}
	l.state.Hue = h
	l.mask |= maskHue
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// SetHex will set the color of the Light to the specified hex string value. This function returns any errors during
// setting the state. This function immediately returns if the 'Manual' attribute is "true" and will change the state
// once the 'Update*'function is called. Hex strings MUST be formalized with at least 6 characters and may begin with a
// '#' symbol. Returns ErrNoColor if the Light does not support color.
func (l *Light) SetHex(h string) error {
	if !l.state.Color {
		return ErrNoColor
	}
	if l.gamut == nil {
		l.gamut = defaultGamut
	}
	x, y, err := xyFromHex(*l.gamut, h)
	if err != nil {
		return err
	}
	return l.SetXY(x, y)
}

// SetEffect will set the light Effect of the Light to the specified value. This function returns any errors during
// setting the state. This function immediately returns if the 'Manual' attribute is "true" and will change the state
// once the 'Update*'function is called.
func (l *Light) SetEffect(e Effect) error {
	l.state.Effect = e
	l.mask |= maskEffect
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// Transition returns the Light state Transition time, represented in a 'time.Duration' object.
func (l Light) Transition() time.Duration {
	return time.Duration(l.state.Transition) * (time.Millisecond * 100)
}

// SetBrightness will set the brightness level of the Light to the specified value. This function returns any
// errors during setting the state. This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*'function is called.
func (l *Light) SetBrightness(b uint8) error {
	l.state.Brightness = b
	l.mask |= maskBrightness
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// SetSaturation will set the light color saturation of the Light to the specified value. This function returns any
// errors during setting the state. This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*'function is called. Returns ErrNoColor if the Light does not support color.
func (l *Light) SetSaturation(s uint8) error {
	if !l.state.Color {
		return ErrNoColor
	}
	l.state.Saturation = s
	l.mask |= maskSaturation
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// SetTransition will set the light state Transition time. This will take effect immediately and will be constant
// until changed with anther call to this function. Setting zero (0) as the argument will make all state changes
// instantaneous. NOTE: Some third-party devices may not handle the transition time correctly and seem to take 1/4th
// of the supplied time.
func (l *Light) SetTransition(t time.Duration) {
	l.state.Transition = uint16(t / (time.Millisecond * 100))
}

// SetTemperature will set the light color temperature of the Light to the specified value. This function returns any
// errors during setting the state. This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*'function is called. Returns ErrNoColor if the Light does not support color.
func (l *Light) SetTemperature(t uint16) error {
	if !l.state.Color {
		return ErrNoColor
	}
	l.state.Temperature = t
	l.mask |= maskTemperature
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// SetXY will set the light color of the Light to the specified CIE 1931 XY value. This function returns any
// errors during setting the state. This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*'function is called. Returns ErrNoColor if the Light does not support color.
func (l *Light) SetXY(x float32, y float32) error {
	if !l.state.Color {
		return ErrNoColor
	}
	l.state.XY[0], l.state.XY[1] = x, y
	l.mask |= maskXY
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// SetCustomPowerOn will change the Light's power on state to a custom value specified. This function returns any
// errors during setting the power on state. This function immediately returns if the 'Manual' attribute is "true"
// and will change the state once the 'Update*' function is called. NOTE: Not every device will support this
// function, mainly only first party (Phillips) devices will have support for this.
func (l *Light) SetCustomPowerOn(s LightState) error {
	l.startup.Mode = startupCustom
	l.startup.Settings = &s.controlState
	l.mask |= maskStartup
	if l.Manual {
		return nil
	}
	return l.UpdateContext(l.bridge.ctx)
}

// SetRGB will set the light color of the Light to the specified RGB value. This function returns any
// errors during setting the state. This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*'function is called. Returns ErrNoColor if the Light does not support color.
func (l *Light) SetRGB(r uint8, g uint8, b uint8) error {
	if !l.state.Color {
		return ErrNoColor
	}
	if l.gamut == nil {
		l.gamut = defaultGamut
	}
	x, y := xyFromRGB(*l.gamut, r, g, b)
	return l.SetXY(x, y)
}
