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

package main

import (
	"fmt"

	"github.com/iDigitalFlame/hue"
)

func main() {

	b, err := hue.Connect("", "")
	if err != nil {
		panic(err)
	}

	l, err := b.Lights()
	if err != nil {
		panic(err)
	}

	for _, v := range l {
		if !v.IsColor() {
			continue
		}
		v.Off()
		v.Manual = true
		v.SetBrightness(0xFF)
		v.SetEffect(hue.EffectColorLoop)
		v.On()
		if err = v.Update(); err != nil {
			panic(err)
		}
	}

	s := b.SensorByName("Hue temperature sensor 1")
	if s == nil {
		panic("no sensor")
	}

	if err := s.SetLed(true); err != nil {
		panic(err)
	}

	t, _ := s.Number("temperature")
	fmt.Println(s.Model, s.Make, s.UUID, "temp", t, "last", s.Updated.String(), s.HasLed(), s.Led(), s.Battery())
}
