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
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/iDigitalFlame/hue"
)

const usage = `huectl - Hue Bridge Controller v1
Copyright (C) 2021 - 2022 iDigitalFlame

Usage:
  huectl -k <API KEY> -a <Hub IP/Address> -t <Target Room/Zone>

  Optional Arguments:
    -list
        List all Groups and Lights that can be targeted. If this is supplied
        no other optional arguments are parsed.
    -on
        Turn on a Light or Control. Takes precedence over "-off".
    -off
        Turn off a Light or Control. Cannot be used with "-on".
    -hex    #XXXXXX
        Set the Light color using a Hex value. This value must be in "#XXXXXX"
        format (6 characters 2 R, 2 G, 2 B), starting with "#".
    -rgb    XXX,XXX,XXX
        Set the Light color using a RGB comma seperated value. This value must
        include the other values, even if they are zero.
    -sat    0 - 255
        Set the Light color saturation using a value from 0 (no saturation) to
        255 (high saturation).
    -temp   0 - 65355
        Set the Light temperature as a value from 0 (cooler) to 65355 (warmer).
    -bright 0 - 255
        Set the Light brightness as a value from 0 (off) to 255 (full brightness).
    -trans  X(s|m|h)
        Use the following duration string as the transition time. This is zero
        (instant) by default.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
`

func main() {
	var (
		trans                       time.Duration
		on, off, list               bool
		bright, sat, temp           int
		target, hex, rgb, addr, key string
		f                           = flag.NewFlagSet("huectl", flag.ExitOnError)
	)
	f.StringVar(&key, "k", "", "")
	f.StringVar(&addr, "a", "", "")
	f.StringVar(&target, "t", "", "")
	f.BoolVar(&on, "on", false, "")
	f.BoolVar(&off, "off", false, "")
	f.BoolVar(&list, "list", false, "")
	f.StringVar(&hex, "hex", "", "")
	f.StringVar(&rgb, "rgb", "", "")
	f.IntVar(&sat, "sat", -1, "")
	f.IntVar(&temp, "temp", -1, "")
	f.IntVar(&bright, "bright", -1, "")
	f.DurationVar(&trans, "trans", 0, "")
	f.Usage = func() {
		os.Stdout.WriteString(usage)
		os.Exit(2)
	}
	if err := f.Parse(os.Args[1:]); err != nil {
		f.Usage()
	}

	if len(key) == 0 {
		os.Stderr.WriteString(`The key value "-k" cannot be empty!` + "\n")
		os.Exit(1)
	}
	if len(addr) == 0 {
		os.Stderr.WriteString(`The address value "-a" cannot be empty!` + "\n")
		os.Exit(1)
	}

	if !list && len(target) == 0 {
		os.Stderr.WriteString(`The target value "-t" cannot be empty!` + "\n")
		os.Exit(1)
	}

	var (
		r, g, b uint64
		err     error
	)
	if len(rgb) > 0 {
		v := strings.Split(strings.TrimSpace(rgb), ",")
		if len(v) != 3 {
			os.Stderr.WriteString(`Invalid RGB value "` + rgb + `"!` + "\n")
			os.Exit(1)
		}
		_ = v[2]
		if r, err = strconv.ParseUint(strings.TrimSpace(v[0]), 10, 8); err != nil {
			os.Stderr.WriteString(`Invalid RGB "R" value "` + rgb + `": ` + err.Error() + "!\n")
			os.Exit(1)
		}
		if g, err = strconv.ParseUint(strings.TrimSpace(v[1]), 10, 8); err != nil {
			os.Stderr.WriteString(`Invalid RGB "G" value "` + rgb + `": ` + err.Error() + "!\n")
			os.Exit(1)
		}
		if b, err = strconv.ParseUint(strings.TrimSpace(v[2]), 10, 8); err != nil {
			os.Stderr.WriteString(`Invalid RGB "B" value "` + rgb + `": ` + err.Error() + "!\n")
			os.Exit(1)
		}
	}

	x, err := hue.Connect(addr, key)
	if err != nil {
		os.Stderr.WriteString(`Failed to connect to "` + addr + `": ` + err.Error() + "!\n")
		os.Exit(1)
	}

	if list {
		g, err := x.Groups()
		if err != nil {
			os.Stderr.WriteString(`Could not get the Groups list from "` + addr + `": ` + err.Error() + "!\n")
			os.Exit(1)
		}
		os.Stdout.WriteString("Groups List\n================\n")
		for k, v := range g {
			os.Stdout.WriteString(
				"[" + k + "] " + v.Name() + ": " + v.Type.String() + " Lights " +
					strconv.FormatUint(uint64(len(v.Lights)), 10) + "\n",
			)
		}

		l, err := x.Lights()
		if err != nil {
			os.Stderr.WriteString(`Could not get the Lights list from "` + addr + `": ` + err.Error() + "!\n")
			os.Exit(1)
		}
		os.Stdout.WriteString("\nLights List\n================\n")
		for k, v := range l {
			if os.Stdout.WriteString("[" + k + "] " + v.Name() + ": " + v.Model + " "); v.IsOn() {
				os.Stdout.WriteString("On Brightness " + strconv.FormatUint(uint64((float32(v.Brightness())/254.0)*100.0), 10) + "% ")
				if v.IsColor() {
					os.Stdout.WriteString(
						"Hue: " + strconv.FormatUint(uint64(v.Hue()), 10) +
							" Sat: " + strconv.FormatUint(uint64(v.Saturation()), 10) +
							" Temp: " + strconv.FormatUint(uint64(v.Temperature()), 10),
					)
				}
				os.Stdout.WriteString("\n")
				continue
			}
			os.Stdout.WriteString("Off\n")
		}

		os.Exit(0)
	}

	i := x.GroupByName(target)
	if i == nil {
		os.Stderr.WriteString(`Could not find target "` + target + `" on "` + addr + `"!` + "\n")
		os.Exit(1)
	}

	for _, e := range i.Controls {
		if e.Manual = true; on {
			e.SetOn(true)
		}
		if !on && off {
			e.SetOn(false)
		}
		if err = e.Update(); err != nil {
			break
		}
	}
	if err != nil {
		os.Stderr.WriteString("Error changing controls: " + err.Error() + "!\n")
		os.Exit(1)
	}

	for _, e := range i.Lights {
		if e.Manual = true; on {
			e.SetOn(true)
		}
		if !on && off {
			e.SetOn(false)
		}
		if bright > -1 {
			e.SetBrightness(uint8(bright))
		}
		if e.SetTransition(trans); !e.IsColor() {
			if err = e.Update(); err != nil {
				break
			}
			continue
		}
		if sat > -1 {
			e.SetSaturation(uint8(sat))
		}
		if temp > -1 {
			e.SetTemperature(uint16(temp))
		}
		if len(rgb) > 0 {
			if err = e.SetRGB(uint8(r), uint8(g), uint8(b)); err != nil {
				break
			}
		}
		if len(hex) > 0 {
			if err = e.SetHex(hex); err != nil {
				break
			}
		}
		if err = e.Update(); err != nil {
			break
		}
	}

	if err != nil {
		os.Stderr.WriteString("Error changing requested lights: " + err.Error() + "!\n")
		os.Exit(1)
	}
	os.Exit(0)
}
