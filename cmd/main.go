package main

import (
	"fmt"

	"github.com/iDigitalFlame/hue"
)

func main() {

	b, err := hue.Connect("<IP>", "<KEY>")
	if err != nil {
		panic(err)
	}

	l := b.SensorByName("Hue temperature sensor 1")
	//if err != nil {
	//	panic(err)
	//}
	/*
		for _, v := range l {
			fmt.Println(
				v.Name(),
				v.Battery(),
			)
			fmt.Println("values", v.Updated)
		}
	*/
	t, _ := l.GetNumber("temperature")
	fmt.Println("temp", t, "last", l.Updated.Time.String(), l.HasLed(), l.Led())

}
