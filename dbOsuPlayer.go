package main

import (
	"github.com/globalsign/mgo/bson"
	"github.com/go-bongo/bongo"
)

// OsuPlayer - A player registered with osu!
type OsuPlayer struct {
	bongo.DocumentBase `bson:",inline"`
	UserID             int      `bson:"userid"`
	PlayerName         string   `bson:"name"`
	LastChecked        int64    `bson:"lastChecked"`
	Modes              OsuModes `bson:"modes"`
}

// OsuModes - A list of the osu! modes
type OsuModes struct {
	Standard OsuModeChecks `bson:"standard"`
	Mania    OsuModeChecks `bson:"mania"`
	Taiko    OsuModeChecks `bson:"taiko"`
	CTB      OsuModeChecks `bson:"ctb"`
}

// OsuModeChecks - Contains an array of objectids with each request made to a player
type OsuModeChecks struct {
	Checks []bson.ObjectId `bson:"checks"`
}
