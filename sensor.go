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

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrNotType is an error returned from the 'Get*' functions when the requested
	// type is not valid for the variable.
	ErrNotType = &errval{s: `requested value type is not valid for this value`}
	// ErrNotFound is an error returned from the 'Get*' functions when the requested
	// name is not reported by the Sensor.
	ErrNotFound = &errval{s: `requested value does not exist`}
)

// Sensor represents a Hue Bridge Sensor Accessory and can be used to read and
// query values.
type Sensor struct {
	Updated sensorTime
	bridge  *Bridge

	Values map[string]interface{}
	config sensorConfig

	ID, Product, UUID string
	name              string

	Make, Model string
	mask        uint16

	Manual bool
}
type sensorTime struct {
	time.Time
}
type sensorConfig struct {
	Led     *bool  `json:"ledindication,omitempty"`
	Battery *uint8 `json:"battery,omitempty"`

	On        bool  `json:"on,omitempty"`
	Alert     Alert `json:"alert,omitempty"`
	Reachable bool  `json:"reachable,omitempty"`
}

// Led will return true if the Sensor's built in Led is on.
func (s *Sensor) Led() bool {
	if s.config.Led == nil {
		return false
	}
	return *s.config.Led
}

// On will switch the Control into the "On" state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (s *Sensor) On() error {
	return s.SetOn(true)
}

// IsOn returns true if this Control is enabled and in the "On" state.
func (s *Sensor) IsOn() bool {
	return s.config.On
}

// Off will switch the Control into the "Off" state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (s *Sensor) Off() error {
	return s.SetOn(false)
}

// Alert returns the Alert status of the Control.
func (s Sensor) Alert() Alert {
	return s.config.Alert
}

// Name returns the Control's display name.
func (s Sensor) Name() string {
	return s.name
}

// HasLed returns true if the Sensor has an onboard LED light.
func (s *Sensor) HasLed() bool {
	return s.config.Led != nil
}

// Update will attempt to sync any changes that have been set while "Manual" is
// set to "true".
//
// This function will return any errors that occur during updating.
func (s *Sensor) Update() error {
	return s.UpdateContext(s.bridge.ctx)
}

// Battery returns the Sensor's battery level.
//
// This function returns 0 if no battery level is reported.
func (s *Sensor) Battery() uint8 {
	if s.config.Battery == nil {
		return 0
	}
	return *s.config.Battery
}

// Reachable returns true if the Control is reachable by the Bridge.
func (s *Sensor) Reachable() bool {
	return s.config.Reachable
}

// HasBattery returns true if the Sensor reports a battery level.
func (s *Sensor) HasBattery() bool {
	return s.config.Battery != nil
}

// SetOn will switch the Control into the specified state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (s *Sensor) SetOn(e bool) error {
	s.config.On = e
	s.mask |= maskOn
	if s.Manual {
		return nil
	}
	return s.UpdateContext(s.bridge.ctx)
}

// SetLed will switch the Sensor's LED light into the specified state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (s *Sensor) SetLed(e bool) error {
	s.config.Led = &e
	s.mask |= maskLed
	if s.Manual {
		return nil
	}
	return s.UpdateContext(s.bridge.ctx)
}

// Contains returns true if the specified value name is returned by the Sensor.
func (s *Sensor) Contains(n string) bool {
	_, ok := s.Values[strings.ToLower(n)]
	return ok
}

// SetAlert will change the Control into the specified Alert state.
//
// This function returns any errors during setting the state.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (s *Sensor) SetAlert(a Alert) error {
	s.config.Alert = a
	s.mask |= maskAlert
	if s.Manual {
		return nil
	}
	return s.UpdateContext(s.bridge.ctx)
}

// SetName will change the Control's display name.
//
// This function returns any errors during setting the display name.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (s *Sensor) SetName(n string) error {
	s.name = n
	s.mask |= maskName
	if s.Manual {
		return nil
	}
	return s.UpdateContext(s.bridge.ctx)
}

// Bool will attempt to retrieve a boolean value from the returned Sensor data.
//
// This will return 'ErrNotFound' if the value name is not found or 'ErrNotType'
// if the specified name does not correlate with a boolean value type.
func (s *Sensor) Bool(n string) (bool, error) {
	return s.GetBool(n)
}

// GetBool will attempt to retrieve a boolean value from the returned Sensor data.
//
// This will return 'ErrNotFound' if the value name is not found or 'ErrNotType'
// if the specified name does not correlate with a boolean value type.
func (s *Sensor) GetBool(n string) (bool, error) {
	v, ok := s.Values[strings.ToLower(n)]
	if !ok {
		return false, ErrNotFound
	}
	var x bool
	if x, ok = v.(bool); ok {
		return x, nil
	}
	return false, ErrNotType
}

// String will attempt to retrieve a string value from the returned Sensor data.
//
// This will return 'ErrNotFound' if the value name is not found or 'ErrNotType'
// if the specified name does not correlate with a string value type.
func (s *Sensor) String(n string) (string, error) {
	return s.GetString(n)
}

// Get will attempt to retrieve a value from the returned Sensor data.
//
// This function returns the data and a boolean which indicates if the value name
// is returned by this Sensor.
func (s *Sensor) Get(n string) (interface{}, bool) {
	v, ok := s.Values[strings.ToLower(n)]
	return v, ok
}

// Number will attempt to retrieve a number value from the returned Sensor data.
//
// This will return 'ErrNotFound' if the value name is not found or 'ErrNotType'
// if the specified name does not corelate with a number value type.
func (s *Sensor) Number(n string) (float64, error) {
	return s.GetNumber(n)
}
func (t *sensorTime) UnmarshalJSON(d []byte) error {
	var (
		s   string
		err error
	)
	if err = json.Unmarshal(d, &s); err != nil {
		return err
	}
	if len(s) == 0 || s == "none" {
		return nil
	}
	if t.Time, err = time.Parse("2006-01-02T15:04:05", s); err != nil {
		return err
	}
	return nil
}

// GetString will attempt to retrieve a string value from the returned Sensor data.
//
// This will return 'ErrNotFound' if the value name is not found or 'ErrNotType'
// if the specified name does not correlate with a string value type.
func (s *Sensor) GetString(n string) (string, error) {
	v, ok := s.Values[strings.ToLower(n)]
	if !ok {
		return "", ErrNotFound
	}
	var x string
	if x, ok = v.(string); ok {
		return x, nil
	}
	return "", ErrNotType
}

// GetNumber will attempt to retrieve a number value from the returned Sensor data.
//
// This will return 'ErrNotFound' if the value name is not found or 'ErrNotType'
// if the specified name does not correlate with a number value type.
func (s *Sensor) GetNumber(n string) (float64, error) {
	v, ok := s.Values[strings.ToLower(n)]
	if !ok {
		return 0, ErrNotFound
	}
	var x float64
	if x, ok = v.(float64); ok {
		return x, nil
	}
	return 0, ErrNotType
}

// UpdateContext will attempt to sync any changes that have been set while
// "Manual" is set to "true".
//
// This function will return any errors that occur during updating.
//
// This function allows a Context to be specified to be used instead of the
// Bridge's base Context.
func (s *Sensor) UpdateContext(x context.Context) error {
	if s.mask == 0 {
		r, err := s.bridge.request(x, http.MethodGet, "/sensors/"+s.ID, nil)
		if err != nil {
			return err
		}
		return s.unmarshal(s.ID, s.bridge, r)
	}
	if s.mask&maskName != 0 {
		b, err := json.Marshal(map[string]string{"name": s.name})
		if err != nil {
			return err
		}
		if _, err = s.bridge.request(x, http.MethodPut, "/sensors/"+s.ID, b); err != nil {
			return err
		}
		if s.mask = s.mask &^ maskName; s.mask == 0 {
			return nil
		}
	}
	b, err := s.config.marshal(s.mask)
	if err != nil {
		return err
	}
	if _, err = s.bridge.request(x, http.MethodPut, "/sensors/"+s.ID+"/config", b); err != nil {
		return err
	}
	s.mask = 0
	return err
}
func (s sensorConfig) marshal(m uint16) ([]byte, error) {
	i := make(map[string]interface{})
	if m&maskOn != 0 {
		i["on"] = s.On
	}
	if m&maskAlert != 0 {
		i["alert"] = s.Alert.String()
	}
	if m&maskLed != 0 && s.Led != nil {
		i["ledindication"] = *s.Led
	}
	d, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return d, nil
}
func (s *Sensor) unmarshal(i string, b *Bridge, d []byte) error {
	var (
		m   map[string]json.RawMessage
		err = json.Unmarshal(d, &m)
	)
	if err != nil {
		return err
	}
	v, ok := m["name"]
	if s.ID, s.bridge = i, b; !ok {
		return &errval{s: `missing "name" parameter value`}
	}
	if err := json.Unmarshal(v, &s.name); err != nil {
		return err
	}
	if v, ok = m["type"]; !ok {
		return &errval{s: `missing "type" parameter value`}
	}
	if err := json.Unmarshal(v, &s.Make); err != nil {
		return err
	}
	if v, ok = m["modelid"]; !ok {
		return &errval{s: `missing "modelid" parameter value`}
	}
	if err := json.Unmarshal(v, &s.Model); err != nil {
		return err
	}
	if v, ok = m["uniqueid"]; ok {
		if err := json.Unmarshal(v, &s.UUID); err != nil {
			return err
		}
	}
	if v, ok = m["productname"]; ok {
		if err := json.Unmarshal(v, &s.Product); err != nil {
			return err
		}
	}
	var a map[string]json.RawMessage
	if v, ok = m["state"]; !ok {
		return &errval{s: `missing "state" parameter value`}
	}
	if err := json.Unmarshal(v, &a); err != nil {
		return err
	}
	if err := json.Unmarshal(v, &s.Values); err != nil {
		return err
	}
	delete(s.Values, "lastupdated")
	if v, ok = a["lastupdated"]; ok {
		if err := json.Unmarshal(v, &s.Updated); err != nil {
			return err
		}
	}
	if v, ok = m["config"]; !ok {
		return &errval{s: `missing "state" parameter value`}
	}
	return json.Unmarshal(v, &s.config)
}
