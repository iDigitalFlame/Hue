package hue

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const timeoutDefault = time.Second * 10

var nothing = context.CancelFunc(func() {})

// Bridge represents a Hue Bridge and can be used to connect and control all the connected devices.
type Bridge struct {
	Timeout time.Duration

	ctx      context.Context
	addr     string
	client   *http.Client
	lights   map[string]*Light
	sensors  map[string]*Sensor
	controls map[string]*Control
}
type errval struct {
	e error
	s string
}
type response []byte

func (b *Bridge) SensorCount() int {
	if len(b.sensors) == 0 {
		b.getSensors(b.ctx)
	}
	return len(b.sensors)
}
func (b *Bridge) Sensor(s string) *Sensor {
	if len(b.sensors) == 0 {
		b.getSensors(b.ctx)
	}
	return b.sensors[s]
}
func (b *Bridge) SensorByName(n string) *Sensor {
	if len(b.sensors) == 0 {
		b.getSensors(b.ctx)
	}
	for _, v := range b.sensors {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
}

func (b *Bridge) LightCount() int {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		b.getControls(b.ctx)
	}
	return len(b.lights)
}
func (b *Bridge) Light(s string) *Light {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		b.getControls(b.ctx)
	}
	return b.lights[s]
}
func (b *Bridge) LightByName(n string) *Light {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		b.getControls(b.ctx)
	}
	for _, v := range b.lights {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
}

func (b *Bridge) ControlCount() int {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		b.getControls(b.ctx)
	}
	return len(b.controls)
}
func (b *Bridge) Control(s string) *Control {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		b.getControls(b.ctx)
	}
	return b.controls[s]
}
func (b *Bridge) ControlByName(n string) *Control {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		b.getControls(b.ctx)
	}
	for _, v := range b.controls {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
}

func (b *Bridge) Update() error {
	return b.UpdateContext(b.ctx)
}
func (b *Bridge) UpdateContext(x context.Context) error {
	b.lights, b.sensors, b.controls = nil, nil, nil
	if err := b.getControls(x); err != nil {
		return err
	}
	if err := b.getSensors(x); err != nil {
		return err
	}
	return nil
}

func (e errval) Error() string {
	if e.e == nil {
		return e.s
	}
	return e.s + ": " + e.e.Error()
}
func (e errval) Unwrap() error {
	return e.e
}
func parse(r string) (*url.URL, error) {
	var (
		i   = strings.IndexRune(r, '/')
		u   *url.URL
		err error
	)
	if i == 0 && len(r) > 2 && r[1] != '/' {
		u, err = url.Parse("/" + r)
	} else if i == -1 || i+1 >= len(r) || r[i+1] != '/' {
		u, err = url.Parse("//" + r)
	} else {
		u, err = url.Parse(r)
	}
	if err != nil {
		return nil, err
	}
	if len(u.Host) == 0 {
		return nil, &errval{s: `bridge: invaid URL "` + r + `" empty host field`}
	}
	if u.Host[len(u.Host)-1] == ':' {
		return nil, &errval{s: `bridge: invalid URL  "` + r + `" invalid port specified`}
	}
	return u, nil
}
func (r *response) UnmarshalJSON(d []byte) error {
	if d[0] == '{' {
		*r = d
		return nil
	}
	var (
		m  []map[string]json.RawMessage
		w  map[string]json.RawMessage
		v  json.RawMessage
		ok bool
	)
	if err := json.Unmarshal(d, &m); err != nil {
		return &errval{s: `bridge: could not unmarshal JSON response`, e: err}
	}
	for i := range m {
		if _, ok = m[i]["success"]; ok {
			continue
		}
		if v, ok = m[i]["error"]; !ok {
			return &errval{s: `bridge: received an invalid JSON response`}
		}
		if err := json.Unmarshal(v, &w); err != nil {
			return &errval{s: `bridge: could not unmarshal JSON response`, e: err}
		}
		var u, e = "unknown URL", "unknown error"
		if v, ok = w["address"]; ok {
			json.Unmarshal(v, &u)
		}
		if v, ok = w["description"]; ok {
			json.Unmarshal(v, &e)
		}
		return &errval{s: `bridge: error returned from "` + u + `": ` + e}
	}
	return nil
}

// Connect returns a Bridge struct based on the specified address/hostname and access key string.
func Connect(address, key string) (*Bridge, error) {
	return ConnectContext(context.Background(), address, key)
}
func (b *Bridge) getSensors(x context.Context) error {
	r, err := b.request(x, http.MethodGet, "/sensors", nil)
	if err != nil {
		return err
	}
	if len(r) == 0 {
		return nil
	}
	m := make(map[string]json.RawMessage)
	if err = json.Unmarshal(r, &m); err != nil {
		return err
	}
	if len(m) == 0 {
		return nil
	}
	b.sensors = make(map[string]*Sensor, len(m))
	for k, v := range m {
		s := new(Sensor)
		if err = s.unmarshal(k, b, v); err != nil {
			return &errval{s: `bridge: could not unmarshal Sensor ID "` + k + `" JSON`, e: err}
		}
		b.sensors[k] = s
	}
	return nil
}

// Lights will attempt to get a list of the connected Lights on the Bridge. This will return an error if there's a
// problem connecting or accessing the Bridge.
func (b *Bridge) Lights() (map[string]*Light, error) {
	return b.LightsContext(b.ctx)
}
func (b *Bridge) getControls(x context.Context) error {
	r, err := b.request(x, http.MethodGet, "/lights", nil)
	if err != nil {
		return err
	}
	if len(r) == 0 {
		return nil
	}
	m := make(map[string]json.RawMessage)
	if err = json.Unmarshal(r, &m); err != nil {
		return err
	}
	if len(m) == 0 {
		return nil
	}
	var d decoder
	b.lights, b.controls = make(map[string]*Light, len(m)), make(map[string]*Control, len(m))
	for k, v := range m {
		if err = d.unmarshal(k, b, v); err != nil {
			return &errval{s: `bridge: could not unmarshal Light ID "` + k + `" JSON`, e: err}
		}
		if d.l != nil {
			b.lights[k] = d.l
			continue
		}
		b.controls[k] = d.c
	}
	return nil
}

// Sensors will attempt to get a list of the connected Sensors on the Bridge. This will return an error if there's
// a problem connecting or accessing the Bridge.
func (b *Bridge) Sensors() (map[string]*Sensor, error) {
	return b.SensorsContext(b.ctx)
}

// Controls will attempt to get a list of the connected Controls (Power/Outlets) on the Bridge. This will return an
// error if there's a problem connecting or accessing the Bridge.
func (b *Bridge) Controls() (map[string]*Control, error) {
	return b.ControlsContext(b.ctx)
}

// LightsContext will attempt to get a list of the connected Lights on the Bridge. This will return an error if
// there's a problem connecting or accessing the Bridge. This function allows for usage of an additional Context
// to be used instead of the Bridge base context.
func (b *Bridge) LightsContext(x context.Context) (map[string]*Light, error) {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		if err := b.getControls(x); err != nil {
			return nil, err
		}
	}
	return b.lights, nil
}

// ConnectContext returns a Bridge struct based on the specified address/hostname and access key string. This
// function allows specifying the base context to be used.
func ConnectContext(x context.Context, address, key string) (*Bridge, error) {
	u, err := parse(address)
	if err != nil {
		return nil, err
	}
	b := &Bridge{
		ctx: x,
		client: &http.Client{
			Timeout: timeoutDefault,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: timeoutDefault, KeepAlive: timeoutDefault, DualStack: true}).DialContext,
				IdleConnTimeout:       timeoutDefault,
				TLSHandshakeTimeout:   timeoutDefault,
				ExpectContinueTimeout: timeoutDefault,
				ResponseHeaderTimeout: timeoutDefault,
			},
		},
		Timeout: timeoutDefault,
	}
	if len(u.Scheme) == 0 {
		u.Scheme = "https"
	}
	if u.Path = ""; u.Scheme == "https" {
		b.client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	s := u.String()
	if s[len(s)-1] != '/' {
		b.addr = s + "/api/" + key
	} else {
		b.addr = s + "api/" + key
	}
	return b, nil
}

// SensorsContext will attempt to get a list of the connected Sensors on the Bridge. This will return an error if
// there's a problem connecting or accessing the Bridge. This function allows for usage of an additional Context
// to be used instead of the Bridge base context.
func (b *Bridge) SensorsContext(x context.Context) (map[string]*Sensor, error) {
	if len(b.sensors) == 0 {
		if err := b.getSensors(x); err != nil {
			return nil, err
		}
	}
	return b.sensors, nil
}

// ControlsContext will attempt to get a list of the connected Controls (Power/Outlets) on the Bridge. This will
// return an error if there's a problem connecting or accessing the Bridge. This function allows for usage of an
// additional Context to be used instead of the Bridge base context.
func (b *Bridge) ControlsContext(x context.Context) (map[string]*Control, error) {
	if len(b.lights) == 0 && len(b.controls) == 0 {
		if err := b.getControls(x); err != nil {
			return nil, err
		}
	}
	return b.controls, nil
}
func (b *Bridge) request(x context.Context, m, u string, d []byte) ([]byte, error) {
	var (
		t = x
		f = nothing
	)
	if b.Timeout > 0 {
		t, f = context.WithTimeout(x, b.Timeout)
	}
	var (
		v, _   = http.NewRequestWithContext(t, m, b.addr+u, bytes.NewReader(d))
		r, err = b.client.Do(v)
	)
	if f(); err != nil {
		return nil, &errval{s: `bridge: could not access "` + b.addr + u + `"`, e: err}
	}
	var o response
	for j := json.NewDecoder(r.Body); j.More(); {
		if err = j.Decode(&o); err != nil {
			return nil, err
		}
	}
	r.Body.Close()
	return o, err
}
