// Copyright (C) 2021 - 2023 iDigitalFlame
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package hue

import "time"

// LightState is a representation of settings that can be used to change the
// state of a LightState.
type LightState struct {
	mask uint16
	controlState
}

// SetAlert will change the LightState into the specified Alert state.
func (s *LightState) SetAlert(a Alert) {
	s.Alert = a
	s.mask |= maskAlert
}

// SetHue will set the color hue of the LightState to the specified value.
func (s *LightState) SetHue(h uint16) {
	s.Hue = h
	s.mask |= maskHue
}

// SetEffect will set the light Effect of the LightState to the specified value.
func (s *LightState) SetEffect(e Effect) {
	s.Effect = e
	s.mask |= maskEffect
}

// SetBrightness will set the brightness level of the LightState to the specified
// value.
func (s *LightState) SetBrightness(b uint8) {
	s.Brightness = b
	s.mask |= maskBrightness
}

// SetSaturation will set the light color saturation of the LightState to the
// specified value.
func (s *LightState) SetSaturation(v uint8) {
	s.Saturation = v
}

// SetHex will set the color of the LightState to the specified hex string value.
//
// Hex strings MUST be formalized with at least 6 characters and may begin with
// a '#' symbol.
func (s *LightState) SetHex(h string) error {
	x, y, err := xyFromHex(*defaultGamut, h)
	if err != nil {
		return err
	}
	s.SetXY(x, y)
	return nil
}

// SetTemperature will set the light color temperature of the LightState to the
// specified value.
func (s *LightState) SetTemperature(t uint16) {
	s.Temperature = t
	s.mask |= maskTemperature
}

// SetXY will set the light color of the LightState to the specified CIE 1931 XY
// value.
func (s *LightState) SetXY(x float32, y float32) {
	s.XY[0], s.XY[1] = x, y
	s.mask |= maskXY
}

// SetTransition will set the light state Transition time. This will take effect
// immediately and will be constant until changed with another call to this
// function.
//
// Setting zero (0) as the argument will make all state changes instantaneous.
//
// NOTE: Some third-party devices may not handle the transition time correctly
// and seem to take 1/4th of the supplied time.
func (s *LightState) SetTransition(t time.Duration) {
	s.Transition = uint16(t / (time.Millisecond * 100))
}

// SetRGB will set the light color of the LightState to the specified RGB value.
func (s *LightState) SetRGB(r uint8, g uint8, b uint8) {
	x, y := xyFromRGB(*defaultGamut, r, g, b)
	s.SetXY(x, y)
}
