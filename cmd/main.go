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
