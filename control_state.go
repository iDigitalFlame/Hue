package hue

import "encoding/json"

const (
	// AlertNone is a light alert effect which the light is not performing an effect.
	AlertNone = Alert(0)
	// AlertSelect is a light alert effect which the light is performing one
	// breathe cycle.
	AlertSelect = Alert(1)
	// AlertBreathe is a light alert effect which the the light is performing
	// breathe cycles for 15 seconds or until an 'AlertNone' command is received.
	//
	// Note that this contains the last alert sent to the light and not its current
	// state. i.e. After the breathe cycle has finished the bridge does not reset
	// the alert to “none“.
	AlertBreathe = Alert(2)

	// EffectNone is a light effect that instructs the light to do nothing.
	EffectNone = Effect(false)
	// EffectColorLoop is a light effect which the light will cycle through all
	// hues using the current brightness and saturation settings.
	EffectColorLoop = Effect(true)

	// StartupResume is a StartupMode option that will resume the previous state
	// of the device when it lost power.
	StartupResume = StartupMode(1)
	// StartupDefault is a StartupMode option that will use the default white
	// color mode when the device resumes from a power loss.
	StartupDefault = StartupMode(0)
	startupCustom  = StartupMode(3)
)

type color bool

// Effect represents the type of light Effect that can be applied to a Hue Control
// object.
type Effect bool

// Alert represents the type of Alert effect that can be applied to a Hue Control
// object.
type Alert uint8
type controlState struct {
	XY          point  `json:"xy,omitempty"`
	Hue         uint16 `json:"hue,omitempty"`
	Transition  uint16 `json:"-"`
	Temperature uint16 `json:"ct,omitempty"`

	On        bool   `json:"on,omitempty"`
	Alert     Alert  `json:"alert,omitempty"`
	Effect    Effect `json:"effect,omitempty"`
	Reachable bool   `json:"reachable,omitempty"`

	Brightness uint8 `json:"bri,omitempty"`
	Saturation uint8 `json:"sat,omitempty"`
	Color      color `json:"colormode,omitempty"`
}

// String returns the name of the Alert effect type.
func (a Alert) String() string {
	switch a {
	case AlertSelect:
		return "select"
	case AlertBreathe:
		return "lselect"
	}
	return "none"
}

// String returns the name of the light Effect type.
func (e Effect) String() string {
	if e {
		return "colorloop"
	}
	return "none"
}

// UnmarshalJSON fulfils the JSON Unmarshaler interface.
func (a *Alert) UnmarshalJSON(d []byte) error {
	if len(d) < 6 || d[0] != '"' {
		return &errval{s: `invalid Alert value`}
	}
	switch d[1] {
	case 'n', 'N':
		*a = AlertNone
	case 's', 'S':
		*a = AlertSelect
	case 'l', 'L':
		*a = AlertBreathe
	default:
		return &errval{s: `invalid Alert value "` + string(d) + `"`}
	}
	return nil
}
func (c *color) UnmarshalJSON(d []byte) error {
	if len(d) < 4 || d[0] != '"' {
		return &errval{s: `invalid Color value`}
	}
	*c = true
	return nil
}

// UnmarshalJSON fulfils the JSON Unmarshaler interface.
func (e *Effect) UnmarshalJSON(d []byte) error {
	if len(d) < 6 || d[0] != '"' {
		return &errval{s: `invalid Effect value`}
	}
	switch d[1] {
	case 'n', 'N':
		*e = EffectNone
	case 'c', 'C':
		*e = EffectColorLoop
	default:
		return &errval{s: `invalid Effect value "` + string(d) + `"`}
	}
	return nil
}
func (s controlState) marshal(m uint16) ([]byte, error) {
	i := make(map[string]interface{})
	if i["transitiontime"] = s.Transition; m&maskOn != 0 {
		i["on"] = s.On
	}
	if m&maskXY != 0 {
		i["xy"] = s.XY
	}
	if m&maskHue != 0 {
		i["hue"] = s.Hue
	}
	if m&maskAlert != 0 {
		i["alert"] = s.Alert.String()
	}
	if m&maskEffect != 0 {
		i["effect"] = s.Effect.String()
	}
	if m&maskBrightness != 0 {
		i["bri"] = s.Brightness
	}
	if m&maskSaturation != 0 {
		i["sat"] = s.Saturation
	}
	if m&maskTemperature != 0 {
		i["ct"] = s.Temperature
	}
	d, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return d, nil
}
