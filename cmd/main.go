package main

import (
	"math/rand"
	"time"

	"github.com/iDigitalFlame/hue"
)

func main() {

	b, err := hue.Connect("<BRIDGE>", "<KEY>")
	if err != nil {
		panic(err)
	}

	l, err := b.Lights()
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	for {
		for _, v := range l {
			if !v.IsColor() {
				continue
			}
			v.On()
			v.Manual = true
			v.SetBrightness(255)
			v.SetEffect(hue.EffectNone)
			v.SetTransition(time.Second * 10)
			v.SetRGB(
				uint8(rand.Int31n(256)), uint8(rand.Int31n(256)), uint8(rand.Int31n(256)),
			)
			v.Update()
			v.Manual = false
		}
		time.Sleep(time.Minute)
	}
}
