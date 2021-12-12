// Copyright (C) 2021 - 2022 iDigitalFlame
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//

package hue

import (
	"context"
	"encoding/json"
	"net/http"
)

const (
	maskXY uint16 = 1 << iota
	maskOn
	maskHue
	maskAlert
	maskEffect
	maskBrightness
	maskSaturation
	maskTemperature
	maskName
	maskStartup
	maskLed
	maskAll = uint16(65535)
)

// Control represents a controllable Hue object. This can be a parent struct for
// a Lights or something that can be toggled, such as an outlet.
type Control struct {
	startup controlStartup
	bridge  *Bridge

	ID, Model, Product string
	name               string

	UUID, Make string
	state      controlState
	mask       uint16

	Manual bool
}
type decoder struct {
	l *Light
	c *Control
}

// StartupMode is a representation of the power on mode of the Control.
type StartupMode uint8
type controlStartup struct {
	Settings *controlState `json:"customsettings,omitempty"`
	Mode     StartupMode   `json:"mode"`
}

// On will switch the Control into the "On" state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (c *Control) On() error {
	return c.SetOn(true)
}

// Off will switch the Control into the "Off" state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (c *Control) Off() error {
	return c.SetOn(false)
}

// IsOn returns true if this Control is enabled and in the "On" state.
func (c *Control) IsOn() bool {
	return c.state.On
}

// Alert returns the Alert status of the Control.
func (c *Control) Alert() Alert {
	return c.state.Alert
}

// Name returns the Control's display name.
func (c *Control) Name() string {
	return c.name
}

// Update will attempt to sync any changes that have been set while "Manual" is
// set to "true". This function will return any errors that occur during updating.
func (c *Control) Update() error {
	return c.UpdateContext(c.bridge.ctx)
}

// Reachable returns true if the Control is reachable by the Bridge.
func (c *Control) Reachable() bool {
	return c.state.Reachable
}

// String returns the name of the power on type.
func (s StartupMode) String() string {
	switch s {
	case StartupDefault:
		return "safety"
	case StartupResume:
		return "powerfail"
	}
	return "custom"
}

// SetOn will switch the Control into the specified state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (c *Control) SetOn(s bool) error {
	c.state.On = s
	if c.mask |= maskOn; c.Manual {
		return nil
	}
	return c.UpdateContext(c.bridge.ctx)
}

// Startup returns the power on method of the Control.
func (c *Control) Startup() StartupMode {
	return c.startup.Mode
}

// SetAlert will change the Control into the specified Alert state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (c *Control) SetAlert(a Alert) error {
	c.state.Alert = a
	if c.mask |= maskAlert; c.Manual {
		return nil
	}
	return c.UpdateContext(c.bridge.ctx)
}

// SetName will change the Control's display name.
//
// This function returns any errors during setting the display name.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (c *Control) SetName(n string) error {
	c.name = n
	if c.mask |= maskName; c.Manual {
		return nil
	}
	return c.UpdateContext(c.bridge.ctx)
}
func (c *Control) marshal() ([]byte, error) {
	m := make(map[string]interface{})
	if c.mask&maskName != 0 {
		m["name"] = c.name
	}
	if c.mask&maskStartup != 0 {
		m["config"] = map[string]interface{}{"startup": c.startup}
	}
	return json.Marshal(m)
}

// SetPowerOn will change the Control's power on state.
//
// This function returns any errors during setting the power on state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
//
// NOTE: Not every device will support this function, mainly only first party
// (Phillips) devices will have support for this.
func (c *Control) SetPowerOn(s StartupMode) error {
	c.startup.Mode, c.startup.Settings = s, nil
	if c.mask |= maskStartup; c.Manual {
		return nil
	}
	return c.UpdateContext(c.bridge.ctx)
}

// MarshalJSON fulfils the JSON Marshaler interface.
func (s StartupMode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON fulfils the JSON Unmarshaler interface.
func (s *StartupMode) UnmarshalJSON(d []byte) error {
	if len(d) < 8 || d[0] != '"' {
		return &errval{s: `invalid StartupMode value`}
	}
	switch d[1] {
	case 'c', 'C':
		*s = startupCustom
	case 's', 'S':
		*s = StartupDefault
	case 'p', 'P':
		*s = StartupResume
	default:
		return &errval{s: `invalid StartupMode value "` + string(d) + `"`}
	}
	return nil
}

// UpdateContext will attempt to sync any changes that have been set while
// "Manual" is set to "true".
//
// This function will return any errors that occur during updating.
//
// This function allows a Context to be specified to be used instead of the
// Bridge's base Context.
func (c *Control) UpdateContext(x context.Context) error {
	if c.mask == 0 {
		r, err := c.bridge.request(x, http.MethodGet, "/lights/"+c.ID, nil)
		if err != nil {
			return err
		}
		var m map[string]json.RawMessage
		if err = json.Unmarshal(r, &m); err != nil {
			return &errval{s: `could not parse response JSON`, e: err}
		}
		return c.unmarshal(m)
	}
	if c.mask >= maskName {
		b, err := c.marshal()
		if c.mask = c.mask &^ maskStartup; err != nil {
			return err
		}
		if _, err = c.bridge.request(x, http.MethodPut, "/lights/"+c.ID, b); err != nil {
			return err
		}
		if c.mask = c.mask &^ maskName; c.mask == 0 {
			return nil
		}
	}
	b, err := c.state.marshal(c.mask)
	if err != nil {
		return err
	}
	if _, err = c.bridge.request(x, http.MethodPut, "/lights/"+c.ID+"/state", b); err != nil {
		return err
	}
	c.mask = 0
	return err
}
func (c *Control) unmarshal(d map[string]json.RawMessage) error {
	v, ok := d["name"]
	if !ok {
		return &errval{s: `missing "name" parameter value`}
	}
	if err := json.Unmarshal(v, &c.name); err != nil {
		return err
	}
	if v, ok = d["uniqueid"]; !ok {
		return &errval{s: `missing "uniqueid" parameter value`}
	}
	if err := json.Unmarshal(v, &c.UUID); err != nil {
		return err
	}
	if v, ok = d["type"]; !ok {
		return &errval{s: `missing "type" parameter value`}
	}
	if err := json.Unmarshal(v, &c.Make); err != nil {
		return err
	}
	if v, ok = d["modelid"]; !ok {
		return &errval{s: `missing "modelid" parameter value`}
	}
	if err := json.Unmarshal(v, &c.Model); err != nil {
		return err
	}
	if v, ok = d["productname"]; !ok {
		return &errval{s: `missing "productname" parameter value`}
	}
	if err := json.Unmarshal(v, &c.Product); err != nil {
		return err
	}
	if v, ok = d["state"]; !ok {
		return &errval{s: `missing "state" parameter value`}
	}
	if err := json.Unmarshal(v, &c.state); err != nil {
		return err
	}
	if v, ok = d["config"]; ok {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(v, &m); err != nil {
			return err
		}
		if v, ok = m["startup"]; ok {
			if err := json.Unmarshal(v, &c.startup); err != nil {
				return err
			}
		}
	}
	return nil
}
func (j *decoder) unmarshal(i string, b *Bridge, d []byte) error {
	var (
		m   map[string]json.RawMessage
		err = json.Unmarshal(d, &m)
	)
	if err != nil {
		return err
	}
	j.c = new(Control)
	if err = j.c.unmarshal(m); err != nil {
		return err
	}
	j.c.bridge, j.c.ID = b, i
	var (
		c     map[string]json.RawMessage
		v, ok = m["capabilities"]
	)
	if !ok {
		return nil
	}
	if err = json.Unmarshal(v, &c); err != nil {
		return err
	}
	if v, ok = c["control"]; !ok {
		return nil
	}
	if err = json.Unmarshal(v, &c); err != nil {
		return err
	}
	if len(v) == 0 || len(c) == 0 {
		return nil
	}
	_, ct := c["ct"]
	if _, ok = c["maxlumen"]; !ok && !ct {
		return nil
	}
	j.l = &Light{Control: *j.c}
	if v, ok = c["colorgamut"]; ok {
		j.l.gamut = new(gamut)
		if err := json.Unmarshal(v, &j.l.gamut); err != nil {
			return err
		}
	}
	return nil
}
