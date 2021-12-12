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
	"encoding/json"
	"math"
	"strconv"
)

var defaultGamut = &gamut{
	Red:   point{0.692, 0.308},
	Blue:  point{0.17, 0.7},
	Green: point{0.153, 0.048},
}

type gamut struct {
	Red, Blue, Green point
}
type point [2]float32

func (g gamut) reachable(x, y float32) bool {
	var (
		a = point{g.Green[0] - g.Red[0], g.Green[1] - g.Red[1]}
		b = point{g.Blue[0] - g.Red[0], g.Blue[1] - g.Red[1]}
		c = point{x - g.Red[0], y - g.Red[1]}
		j = (c[0]*b[1] - c[1]*b[0]) / (a[0]*b[1] - a[1]*b[0])
		k = (a[0]*c[1] - a[1]*c[0]) / (a[0]*b[1] - a[1]*b[0])
	)
	return j >= 0 && k >= 0 && j+k <= 1
}
func (g *gamut) UnmarshalJSON(d []byte) error {
	var c []json.RawMessage
	if err := json.Unmarshal(d, &c); err != nil {
		return err
	}
	if len(c) != 3 {
		return &errval{s: `json: invalid Color Gamut value`}
	}
	if err := json.Unmarshal(c[0], &g.Red); err != nil {
		return err
	}
	if err := json.Unmarshal(c[1], &g.Blue); err != nil {
		return err
	}
	if err := json.Unmarshal(c[2], &g.Green); err != nil {
		return err
	}
	return nil
}
func closest(a, b point, x, y float32) (float32, float32) {
	var (
		h = point{x - a[0], y - a[1]}
		j = point{b[0] - a[0], b[1] - a[1]}
		k = (h[0]*j[0] + h[1]*j[1]) / (j[0]*j[0] + j[1]*j[1])
	)
	if k < 0 {
		k = 0
	} else if k > 1 {
		k = 1
	}
	return a[0] + j[0]*k, a[1] + j[1]*k
}
func xyFromHex(c gamut, s string) (float32, float32, error) {
	if len(s) < 6 || len(s) > 7 || (len(s) == 7 && s[0] != '#') {
		return 0, 0, &errval{s: `hex value "` + s + `" is invalid`}
	}
	i := 0
	if s[0] == '#' {
		i++
	}
	var (
		r, g, b uint64
		err     error
	)
	if r, err = strconv.ParseUint(s[i:i+2], 16, 16); err != nil {
		return 0, 0, &errval{s: `hex red value is invalid`, e: err}
	}
	if g, err = strconv.ParseUint(s[i+2:i+4], 16, 16); err != nil {
		return 0, 0, &errval{s: `hex green value is invalid`, e: err}
	}
	if b, err = strconv.ParseUint(s[i+4:i+6], 16, 16); err != nil {
		return 0, 0, &errval{s: `hex blue value is invalid`, e: err}
	}
	x, y := xyFromRGB(c, uint8(r), uint8(g), uint8(b))
	return x, y, nil
}
func (g gamut) closestPoint(x, y float32) (float32, float32) {
	var (
		ax, ay = closest(g.Red, g.Green, x, y)
		bx, by = closest(g.Blue, g.Red, x, y)
		cx, cy = closest(g.Green, g.Blue, x, y)
		ad     = float32(math.Sqrt(float64((x-ax)*(x-ax) + (y-ay)*(y-ay))))
		bd     = float32(math.Sqrt(float64((x-bx)*(x-bx) + (y-by)*(y-by))))
		cd     = float32(math.Sqrt(float64((x-cx)*(x-cx) + (y-cy)*(y-cy))))
		l      = ad
		fx     = ax
		fy     = ay
	)
	if bd < l {
		l, fx, fy = bd, bx, by
	}
	if cd < l {
		return cx, cy
	}
	return fx, fy
}
func rgbFromXy(c gamut, l, x, y float32) (uint8, uint8, uint8) {
	sx, sy := x, y
	if !c.reachable(sx, sy) {
		sx, sy = c.closestPoint(sx, sy)
	}
	var (
		cx = (l / sy) * sx
		cz = (l / sy) * (1 - sx - sy)
		r  = cx*1.656492 - l*0.354851 - cz*0.255038
		g  = -cx*0.707196 + l*1.655397 + cz*0.036152
		b  = cx*0.051713 - l*0.121364 + cz*1.011530
	)
	if r <= 0.0031308 {
		r = 12.92 * r
	} else {
		r = (1.0+0.055)*float32(math.Pow(float64(r), 1.0/2.4)) - 0.055
	}
	if g <= 0.0031308 {
		g = 12.92 * g
	} else {
		g = (1.0+0.055)*float32(math.Pow(float64(g), 1.0/2.4)) - 0.055
	}
	if b <= 0.0031308 {
		b = 12.92 * b
	} else {
		b = (1.0+0.055)*float32(math.Pow(float64(b), 1.0/2.4)) - 0.055
	}
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}
	if r > 1 || b > 1 || g > 1 {
		m := r
		if b > m {
			m = b
		}
		if g > m {
			m = g
		}
		r, g, b = r/m, g/m, b/m
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
func xyFromRGB(c gamut, red, green, blue uint8) (float32, float32) {
	r, g, b := float64(red)/255.0, float64(green)/255.0, float64(blue)/255.0
	if r > 0.04045 {
		r = math.Pow(((r + 0.055) / (1.0 + 0.055)), 2.4)
	} else {
		r = r / 12.92
	}
	if g > 0.04045 {
		g = math.Pow(((g + 0.055) / (1.0 + 0.055)), 2.4)
	} else {
		g = g / 12.92
	}
	if b > 0.04045 {
		b = math.Pow(((b + 0.055) / (1.0 + 0.055)), 2.4)
	} else {
		b = b / 12.92
	}
	var (
		x  = r*0.664511 + g*0.154324 + b*0.162028
		y  = r*0.283881 + g*0.668433 + b*0.047685
		z  = r*0.000088 + g*0.072310 + b*0.986039
		cx = float32(x / (x + y + z))
		cy = float32(y / (x + y + z))
	)
	if c.reachable(cx, cy) {
		return cx, cy
	}
	return c.closestPoint(cx, cy)
}
