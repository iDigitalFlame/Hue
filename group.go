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
)

const (
	// Luminaire is a lighting installation of default groupings of hue lights.
	//
	// The bridge will pre-install these groups for ease of use.
	// This type cannot be created manually.
	//
	// Also, a light can only be in a maximum of one luminaire group. See
	// multisource luminaires for more info.
	Luminaire groupType = 0
	// LightSource is a group of lights which is created by the bridge based on
	// multisource luminaire attributes of Zigbee light resource.
	LightSource groupType = iota
	// LightGroup is a group of lights that can be controlled together.
	//
	// This the default group type that the bridge generates for user created
	// groups. Default type when no type is given on creation.
	LightGroup
	// Room is a group of lights that are physically located in the same place
	// in the house. Rooms behave similar as light groups, except:
	//
	// (1) A room can be empty and contain 0 lights,
	// (2) a light is only allowed in one room and
	// (3) a room isn’t automatically deleted when all lights in that room are
	//     deleted.
	Room
	// Entertainment groups describe a group of lights that are used in an
	// entertainment setup. Locations describe the relative position of the lights
	// in an entertainment setup. E.g. for TV the position is relative to the TV.
	// Can be used to configure streaming sessions. Entertainment groups behave
	// in a similar way as light groups, with the exception: it can be empty and
	// contain 0 lights.
	//
	// The group is also not automatically recycled when lights are deleted.
	//
	// The group of lights can be controlled together as in LightGroup.
	Entertainment
	// Zone types describe a group of lights that can be controlled together.
	// Zones can be empty and contain 0 lights.
	//
	// A light is allowed to be in multiple zones.
	Zone
	// All is a special group containing all lights in the system, and is not
	// returned by the ‘get all groups’ command. This group is not visible, and
	// cannot be created, modified or deleted using the API.
	All
)

// Room Class Type Constants
const (
	ClassInvalid GroupClass = 0
	ClassAttic   GroupClass = iota
	ClassBalcony
	ClassBarbecue
	ClassBathroom
	ClassBedroom
	ClassCarport
	ClassCloset
	ClassComputer
	ClassDining
	ClassDownstairs
	ClassDriveway
	ClassFrontDoor
	ClassGarage
	ClassGarden
	ClassGuestRoom
	ClassGym
	ClassHallway
	ClassHome
	ClassKidsBedroom
	ClassKitchen
	ClassLaundryRoom
	ClassLivingRoom
	ClassLounge
	ClassManCave
	ClassMusic
	ClassNursery
	ClassOffice
	ClassOther
	ClassPool
	ClassPorch
	ClassReading
	ClassRecreation
	ClassStaircase
	ClassStorage
	ClassStudio
	ClassTV
	ClassTerrace
	ClassToilet
	ClassTopFloor
	ClassUpstairs
)

// Group is a struct that can be used to access and control the Sensors, Lights
// and Controls included. Groups can be used to control multiple devices at a
// single time.
type Group struct {
	bridge *Bridge

	ID   string
	name string

	Lights   []*Light
	Sensors  []*Sensor
	Controls []*Control
	action   controlState
	mask     uint16

	On, AllOn, Manual bool

	Type  groupType
	class GroupClass
}
type groupType uint8

// GroupClass is an integer representation that is used to represent the Group
// classification and can be used to determine the display icon in the Hue app.
type GroupClass uint8

// Name returns the name of the Group.
func (g *Group) Name() string {
	return g.name
}

// Class returns the Group Class type.
func (g *Group) Class() GroupClass {
	return g.class
}
func (t groupType) String() string {
	switch t {
	case All:
		return "LightGroup"
	case Room:
		return "Room"
	case Zone:
		return "Zone"
	case Luminaire:
		return "Luminaire"
	case LightSource:
		return "Lightsource"
	case LightGroup:
		return "LightGroup"
	case Entertainment:
		return "Entertainment"
	}
	return "Room"
}
func (r GroupClass) String() string {
	switch r {
	case ClassInvalid:
		return "Auto/Invalid"
	case ClassAttic:
		return "Attic"
	case ClassBalcony:
		return "Balcony"
	case ClassBarbecue:
		return "Barbecue"
	case ClassBathroom:
		return "Bathroom"
	case ClassBedroom:
		return "Bedroom"
	case ClassCarport:
		return "Carport"
	case ClassCloset:
		return "Closet"
	case ClassComputer:
		return "Computer"
	case ClassDining:
		return "Dining"
	case ClassDownstairs:
		return "Downstairs"
	case ClassDriveway:
		return "Driveway"
	case ClassFrontDoor:
		return "Front door"
	case ClassGarage:
		return "Garage"
	case ClassGarden:
		return "Garden"
	case ClassGuestRoom:
		return "Guest room"
	case ClassGym:
		return "Gym"
	case ClassHallway:
		return "Hallway"
	case ClassHome:
		return "Home"
	case ClassKidsBedroom:
		return "Kids bedroom"
	case ClassKitchen:
		return "Kitchen"
	case ClassLaundryRoom:
		return "Laundry room"
	case ClassLivingRoom:
		return "Living room"
	case ClassLounge:
		return "Lounge"
	case ClassManCave:
		return "Man cave"
	case ClassMusic:
		return "Music"
	case ClassNursery:
		return "Nursery"
	case ClassOffice:
		return "Office"
	case ClassOther:
		return "Other"
	case ClassPool:
		return "Pool"
	case ClassPorch:
		return "Porch"
	case ClassReading:
		return "Reading"
	case ClassRecreation:
		return "Recreation"
	case ClassStaircase:
		return "Staircase"
	case ClassStorage:
		return "Storage"
	case ClassStudio:
		return "Studio"
	case ClassTV:
		return "TV"
	case ClassTerrace:
		return "Terrace"
	case ClassToilet:
		return "Toilet"
	case ClassTopFloor:
		return "Top floor"
	case ClassUpstairs:
		return "Upstairs"
	}
	return "Other"
}

// SetName will change the Group's display name. This function returns any errors
// during setting the display name.
//
// This function immediately returns if the 'Manual' attribute is "true" and will
// change the state once the 'Update*' function is called.
func (g *Group) SetName(n string) error {
	g.name = n
	if g.mask |= maskName; g.Manual {
		return nil
	}
	return g.UpdateContext(g.bridge.ctx)
}
func (t groupType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

// MarshalJSON converts the GroupClass into a JSON byte array.
func (r GroupClass) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}
func (t *groupType) UnmarshalJSON(d []byte) error {
	if len(d) < 6 || d[0] != '"' {
		return &errval{s: `invalid groupType value`}
	}
	switch {
	case d[1] == 'R':
		*t = Room
	case d[1] == 'Z':
		*t = Zone
	case d[1] == 'E':
		*t = Entertainment
	case d[1] == 'L' && d[3] == 'm':
		*t = Luminaire
	case len(d) < 8:
		return &errval{s: `invalid groupType value`}
	case d[1] == 'L' && d[6] == 's':
		*t = LightSource
	case d[1] == 'L' && d[6] == 'G':
		*t = LightGroup
	}
	return nil
}

// UnmarshalJSON will take the JSON byte array and convert it into a GroupClass
// instance and sets the value on this instance.
func (r *GroupClass) UnmarshalJSON(d []byte) error {
	if len(d) < 4 || d[0] != '"' {
		return &errval{s: `invalid GroupClass value`}
	}
	switch {
	case d[1] == 'T' && d[2] == 'V':
		*r = ClassTV
	case d[1] == 'G' && d[3] == 'm':
		*r = ClassGym
	case d[1] == 'A' && d[3] == 't':
		*r = ClassAttic
	case d[1] == 'B' && d[3] == 'l':
		*r = ClassBalcony
	case d[1] == 'B' && d[3] == 'r':
		*r = ClassBarbecue
	case d[1] == 'B' && d[3] == 't':
		*r = ClassBathroom
	case d[1] == 'B' && d[3] == 'd':
		*r = ClassBedroom
	case d[1] == 'C' && d[3] == 'r':
		*r = ClassCarport
	case d[1] == 'C' && d[3] == 'o':
		*r = ClassCloset
	case d[1] == 'C' && d[3] == 'm':
		*r = ClassComputer
	case d[1] == 'D' && d[3] == 'n':
		*r = ClassDining
	case d[1] == 'D' && d[3] == 'w':
		*r = ClassDownstairs
	case d[1] == 'D' && d[3] == 'i':
		*r = ClassDriveway
	case d[1] == 'F' && d[3] == 'o':
		*r = ClassFrontDoor
	case d[1] == 'G' && d[4] == 'a':
		*r = ClassGarage
	case d[1] == 'G' && d[4] == 'd':
		*r = ClassGarden
	case d[1] == 'G' && d[3] == 'e':
		*r = ClassGuestRoom
	case d[1] == 'H' && d[3] == 'l':
		*r = ClassHallway
	case d[1] == 'H' && d[3] == 'm':
		*r = ClassHome
	case d[1] == 'K' && d[3] == 'd':
		*r = ClassKidsBedroom
	case d[1] == 'K' && d[3] == 't':
		*r = ClassKitchen
	case d[1] == 'L' && d[4] == 'n':
		*r = ClassLaundryRoom
	case d[1] == 'L' && d[4] == 'i':
		*r = ClassLivingRoom
	case d[1] == 'L' && d[2] == 'o':
		*r = ClassLounge
	case d[1] == 'M' && d[3] == 'n':
		*r = ClassManCave
	case d[1] == 'M' && d[3] == 's':
		*r = ClassMusic
	case d[1] == 'N' && d[3] == 'r':
		*r = ClassNursery
	case d[1] == 'O' && d[3] == 'f':
		*r = ClassOffice
	case d[1] == 'O' && d[3] == 'h':
		*r = ClassOther
	case d[1] == 'P' && d[3] == 'o':
		*r = ClassPool
	case d[1] == 'P' && d[3] == 'r':
		*r = ClassPorch
	case d[1] == 'R' && d[3] == 'a':
		*r = ClassReading
	case d[1] == 'R' && d[3] == 'c':
		*r = ClassRecreation
	case d[1] == 'S' && d[3] == 'a':
		*r = ClassStaircase
	case d[1] == 'S' && d[3] == 'o':
		*r = ClassStorage
	case d[1] == 'S' && d[3] == 'u':
		*r = ClassStudio
	case d[1] == 'T' && d[3] == 'r':
		*r = ClassTerrace
	case d[1] == 'T' && d[3] == 'i':
		*r = ClassToilet
	case d[1] == 'T' && d[3] == 'p':
		*r = ClassTopFloor
	case d[1] == 'U' && d[3] == 's':
		*r = ClassUpstairs
	}
	return nil
}

// UpdateContext will attempt to sync any changes that have been set while
// "Manual" is set to "true".
//
// This function will return any errors that occur during updating.
//
// This function allows a Context to be specified to be used instead of the
// Bridge's base Context.
func (g *Group) UpdateContext(x context.Context) error {
	if g.mask == 0 {
		r, err := g.bridge.request(x, http.MethodGet, "/groups/"+g.ID, nil)
		if err != nil {
			return err
		}
		return g.unmarshal(g.ID, g.bridge, r)
	}
	if g.mask&maskName != 0 {
		b, err := json.Marshal(map[string]string{"name": g.name})
		if err != nil {
			return err
		}
		if _, err = g.bridge.request(x, http.MethodPut, "/groups/"+g.ID, b); err != nil {
			return err
		}
		if g.mask = g.mask &^ maskName; g.mask == 0 {
			return nil
		}
	}
	b, err := g.action.marshal(g.mask)
	if err != nil {
		return err
	}
	if _, err = g.bridge.request(x, http.MethodPut, "/groups/"+g.ID+"/action", b); err != nil {
		return err
	}
	g.mask = 0
	return err
}
func (g *Group) unmarshal(i string, b *Bridge, d []byte) error {
	var (
		m   map[string]json.RawMessage
		err = json.Unmarshal(d, &m)
	)
	if err != nil {
		return err
	}
	v, ok := m["name"]
	if g.ID, g.bridge = i, b; !ok {
		return &errval{s: `json: missing "name" parameter value`}
	}
	if err := json.Unmarshal(v, &g.name); err != nil {
		return err
	}
	if v, ok = m["type"]; !ok {
		return &errval{s: `json: missing "type" parameter value`}
	}
	if err := json.Unmarshal(v, &g.Type); err != nil {
		return err
	}
	if v, ok = m["action"]; ok {
		if err := json.Unmarshal(v, &g.action); err != nil {
			return err
		}
	}
	if v, ok = m["class"]; ok {
		if err := json.Unmarshal(v, &g.class); err != nil {
			return err
		}
	}
	var s []string
	if v, ok = m["lights"]; ok && len(v) > 4 {
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		g.Controls, g.Lights = make([]*Control, 0), make([]*Light, 0, len(s))
		for i := range s {
			if x, ok2 := b.lights[s[i]]; ok2 {
				g.Lights = append(g.Lights, x)
				continue
			}
			if x, ok2 := b.controls[s[i]]; ok2 {
				g.Controls = append(g.Controls, x)
			}
		}
	}
	if v, ok = m["sensors"]; ok && len(v) > 4 {
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		g.Sensors = make([]*Sensor, 0, len(s))
		for i := range s {
			if x, ok2 := b.sensors[s[i]]; ok2 {
				g.Sensors = append(g.Sensors, x)
			}
		}
	}
	if v, ok = m["state"]; ok {
		if err := json.Unmarshal(v, &m); err != nil {
			return err
		}
		if x, ok2 := m["any_on"]; ok2 {
			if err := json.Unmarshal(x, &g.On); err != nil {
				return err
			}
		}
		if x, ok2 := m["all_on"]; ok2 {
			if err := json.Unmarshal(x, &g.AllOn); err != nil {
				return err
			}
		}
	}
	return nil
}
