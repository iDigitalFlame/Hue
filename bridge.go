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
	"sync"
	"time"
)

const timeoutDefault = time.Second * 10

var nothing = context.CancelFunc(func() {})

// Bridge represents a Hue Bridge and can be used to connect and control all the connected devices.
type Bridge struct {
	Timeout time.Duration

	all      *Group
	ctx      context.Context
	addr     string
	lock     sync.Mutex
	client   *http.Client
	groups   map[string]*Group
	lights   map[string]*Light
	sensors  map[string]*Sensor
	controls map[string]*Control
}
type errval struct {
	e error
	s string
}
type response []byte

func (e errval) Error() string {
	if e.e == nil {
		return e.s
	}
	return e.s + ": " + e.e.Error()
}
func (e errval) Unwrap() error {
	return e.e
}

// Update will fetch updates to all the devices exposed by the Bridge. This function will refresh and add
// any new devices and remove deleted ones.
func (b *Bridge) Update() error {
	return b.UpdateContext(b.ctx)
}

// LightCount returns the number of Lights connected to the Bridge.
func (b *Bridge) LightCount() int {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		b.getControls(b.ctx)
		b.lock.Unlock()
	}
	return len(b.lights)
}

// SensorCount returns the number of Sensors connected to the Bridge.
func (b *Bridge) SensorCount() int {
	if b.sensors == nil {
		b.lock.Lock()
		b.getSensors(b.ctx)
		b.lock.Unlock()
	}
	return len(b.sensors)
}

// ControlCount returns the number of Controls connected to the Bridge.
func (b *Bridge) ControlCount() int {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		b.getControls(b.ctx)
		b.lock.Unlock()
	}
	return len(b.controls)
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
		return nil, &errval{s: `invaid URL "` + r + `" empty host field`}
	}
	if u.Host[len(u.Host)-1] == ':' {
		return nil, &errval{s: `invalid URL  "` + r + `" invalid port specified`}
	}
	return u, nil
}

// All will return the 'All' Group. This group is a special group that automatically contains every light,
// control and sensor connected to the Bridge.
func (b *Bridge) All() (*Group, error) {
	return b.AllContext(b.ctx)
}

// Light returns a Light by the ID string. This function returns nil if there is no Light with that ID.
func (b *Bridge) Light(s string) *Light {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		b.getControls(b.ctx)
		b.lock.Unlock()
	}
	return b.lights[s]
}

// Group returns a Group by the ID string. This function returns nil if there is no Group with that ID.
func (b *Bridge) Group(s string) *Group {
	if b.groups == nil {
		b.lock.Lock()
		b.getGroups(b.ctx)
		b.lock.Unlock()
	}
	return b.groups[s]
}

// Sensor returns a Sensor by the ID string. This function returns nil if there is no Sensor with that ID.
func (b *Bridge) Sensor(s string) *Sensor {
	if b.sensors == nil {
		b.lock.Lock()
		b.getSensors(b.ctx)
		b.lock.Unlock()
	}
	return b.sensors[s]
}

// Control returns a Control by the ID string. This function returns nil if there is no Control with that ID.
func (b *Bridge) Control(s string) *Control {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		b.getControls(b.ctx)
		b.lock.Unlock()
	}
	return b.controls[s]
}

// GroupByName returns a Group by the Name string. This function returns nil if there is no Group with that Name.
func (b *Bridge) GroupByName(n string) *Group {
	if b.groups == nil {
		b.lock.Lock()
		b.getGroups(b.ctx)
		b.lock.Unlock()
	}
	for _, v := range b.groups {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
}

// LightByName returns a Light by the Name string. This function returns nil if there is no Light with that Name.
func (b *Bridge) LightByName(n string) *Light {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		b.getControls(b.ctx)
		b.lock.Unlock()
	}
	for _, v := range b.lights {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
}

// SensorByName returns a Sensor by the Name string. This function returns nil if there is no Sensor with that Name.
func (b *Bridge) SensorByName(n string) *Sensor {
	if b.sensors == nil {
		b.lock.Lock()
		b.getSensors(b.ctx)
		b.lock.Unlock()
	}
	for _, v := range b.sensors {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
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
		return &errval{s: `could not unmarshal JSON response`, e: err}
	}
	for i := range m {
		if _, ok = m[i]["success"]; ok {
			continue
		}
		if v, ok = m[i]["error"]; !ok {
			return &errval{s: `received an invalid JSON response`}
		}
		if err := json.Unmarshal(v, &w); err != nil {
			return &errval{s: `could not unmarshal JSON response`, e: err}
		}
		var u, e = "unknown URL", "unknown error"
		if v, ok = w["address"]; ok {
			json.Unmarshal(v, &u)
		}
		if v, ok = w["description"]; ok {
			json.Unmarshal(v, &e)
		}
		return &errval{s: `error returned from "` + u + `": ` + e}
	}
	return nil
}

// ControlByName returns a Control by the Name string. This function returns nil if there is no Control with that Name.
func (b *Bridge) ControlByName(n string) *Control {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		b.getControls(b.ctx)
		b.lock.Unlock()
	}
	for _, v := range b.controls {
		if strings.EqualFold(n, v.name) {
			return v
		}
	}
	return nil
}

// Connect returns a Bridge struct based on the specified address/hostname and access key string.
func Connect(address, key string) (*Bridge, error) {
	return ConnectContext(context.Background(), address, key)
}
func (b *Bridge) getGroups(x context.Context) error {
	if b.lights == nil || b.controls == nil {
		if err := b.getControls(x); err != nil {
			return err
		}
	}
	if b.sensors == nil {
		if err := b.getSensors(x); err != nil {
			return err
		}
	}
	r, err := b.request(x, http.MethodGet, "/groups", nil)
	if err != nil || len(r) == 0 {
		return err
	}
	m := make(map[string]json.RawMessage)
	if err = json.Unmarshal(r, &m); err != nil || len(m) == 0 {
		return err
	}
	b.groups = make(map[string]*Group, len(m))
	for k, v := range m {
		g := new(Group)
		if err = g.unmarshal(k, b, v); err != nil {
			return &errval{s: `could not unmarshal Group ID "` + k + `" JSON`, e: err}
		}
		b.groups[k] = g
	}
	return nil
}
func (b *Bridge) getSensors(x context.Context) error {
	r, err := b.request(x, http.MethodGet, "/sensors", nil)
	if err != nil || len(r) == 0 {
		return err
	}
	m := make(map[string]json.RawMessage)
	if err = json.Unmarshal(r, &m); err != nil || len(m) == 0 {
		return err
	}
	b.sensors = make(map[string]*Sensor, len(m))
	for k, v := range m {
		s := new(Sensor)
		if err = s.unmarshal(k, b, v); err != nil {
			return &errval{s: `could not unmarshal Sensor ID "` + k + `" JSON`, e: err}
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

// Groups will attempt to get a list of the Groups on the Bridge. This will return an error if there's a
// problem connecting or accessing the Bridge. This function will also pull all the connected Sensors, Lights and
// Controls connected to the Bridge in order to link them properly.
func (b *Bridge) Groups() (map[string]*Group, error) {
	return b.GroupsContext(b.ctx)
}
func (b *Bridge) getGroupAll(x context.Context) error {
	if b.lights == nil || b.controls == nil {
		if err := b.getControls(x); err != nil {
			return err
		}
	}
	if b.sensors == nil {
		if err := b.getSensors(x); err != nil {
			return err
		}
	}
	r, err := b.request(x, http.MethodGet, "/groups/0", nil)
	if err != nil || len(r) == 0 {
		return err
	}
	b.all = new(Group)
	if err = b.all.unmarshal("0", b, r); err != nil {
		return &errval{s: `could not unmarshal All Group JSON`, e: err}
	}
	b.all.Type = All
	return nil
}
func (b *Bridge) getControls(x context.Context) error {
	r, err := b.request(x, http.MethodGet, "/lights", nil)
	if err != nil || len(r) == 0 {
		return err
	}
	m := make(map[string]json.RawMessage)
	if err = json.Unmarshal(r, &m); err != nil || len(m) == 0 {
		return err
	}
	var d decoder
	b.lights, b.controls = make(map[string]*Light, len(m)), make(map[string]*Control, len(m))
	for k, v := range m {
		if err = d.unmarshal(k, b, v); err != nil {
			return &errval{s: `could not unmarshal Light ID "` + k + `" JSON`, e: err}
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

// UpdateContext will fetch updates to all the devices exposed by the Bridge. This function will refresh and add
// any new devices and remove deleted ones. This function allows for specifying a Context to be used instead of
// the Bridge base context.
func (b *Bridge) UpdateContext(x context.Context) error {
	b.lock.Lock()
	b.all, b.lights, b.sensors, b.controls = nil, nil, nil, nil
	if err := b.getControls(x); err != nil {
		b.lock.Unlock()
		return err
	}
	if err := b.getSensors(x); err != nil {
		b.lock.Unlock()
		return err
	}
	err := b.getGroups(x)
	b.lock.Unlock()
	return err
}

// Controls will attempt to get a list of the connected Controls (Power/Outlets) on the Bridge. This will return an
// error if there's a problem connecting or accessing the Bridge.
func (b *Bridge) Controls() (map[string]*Control, error) {
	return b.ControlsContext(b.ctx)
}

// AllContext will return the 'All' Group. This group is a special group that automatically contains every light,
// control and sensor connected to the Bridge. This function allows for specifying a Context to be used instead of
// the Bridge base context.
func (b *Bridge) AllContext(x context.Context) (*Group, error) {
	if b.all == nil {
		b.lock.Lock()
		err := b.getGroupAll(x)
		b.lock.Unlock()
		return b.all, err
	}
	return b.all, nil
}

// LightsContext will attempt to get a list of the connected Lights on the Bridge. This will return an error if
// there's a problem connecting or accessing the Bridge. This function allows for usage of an additional Context
// to be used instead of the Bridge base context.
func (b *Bridge) LightsContext(x context.Context) (map[string]*Light, error) {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		err := b.getControls(x)
		b.lock.Unlock()
		return b.lights, err
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

// GroupsContext will attempt to get a list of the Groups on the Bridge. This will return an error if there's a
// problem connecting or accessing the Bridge. This function will also pull all the connected Sensors, Lights and
// Controls connected to the Bridge in order to link them properly. This function allows for usage of an additional
// Context to be used instead of the Bridge base context.
func (b *Bridge) GroupsContext(x context.Context) (map[string]*Group, error) {
	if b.groups == nil {
		b.lock.Lock()
		err := b.getGroups(x)
		b.lock.Unlock()
		return b.groups, err
	}
	return b.groups, nil
}

// SensorsContext will attempt to get a list of the connected Sensors on the Bridge. This will return an error if
// there's a problem connecting or accessing the Bridge. This function allows for usage of an additional Context
// to be used instead of the Bridge base context.
func (b *Bridge) SensorsContext(x context.Context) (map[string]*Sensor, error) {
	if b.sensors == nil {
		b.lock.Lock()
		err := b.getSensors(x)
		b.lock.Unlock()
		return b.sensors, err
	}
	return b.sensors, nil
}

// ControlsContext will attempt to get a list of the connected Controls (Power/Outlets) on the Bridge. This will
// return an error if there's a problem connecting or accessing the Bridge. This function allows for usage of an
// additional Context to be used instead of the Bridge base context.
func (b *Bridge) ControlsContext(x context.Context) (map[string]*Control, error) {
	if b.lights == nil || b.controls == nil {
		b.lock.Lock()
		err := b.getControls(x)
		b.lock.Unlock()
		return b.controls, err
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
	if err != nil {
		f()
		return nil, &errval{s: `could not access "` + b.addr + u + `"`, e: err}
	}
	var o response
	for j := json.NewDecoder(r.Body); j.More(); {
		if err = j.Decode(&o); err != nil {
			break
		}
	}
	f()
	r.Body.Close()
	return o, err
}
