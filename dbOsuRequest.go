package main

import (
	"github.com/globalsign/mgo/bson"
	"github.com/go-bongo/bongo"
)

// OsuRequest - Holds information regarding data we obtain when requesting an osu! players data
type OsuRequest struct {
	bongo.DocumentBase `bson:",inline"`
	OsuPlayer          bson.ObjectId  `bson:"player"`
	DateChecked        int            `bson:"dateChecked"`
	Data               OsuRequestData `bson:"data"`
}

// OsuRequestData - The data we get from the osu! api
type OsuRequestData struct {
	PlayerID   string            `json:"id" bson:"id"`
	PlayerName string            `json:"name" bson:"name"`
	Counts     requestDataCounts `json:"counts" bson:"counts"`
	Scores     requestDataScores `json:"scores" bson:"scores"`
	PP         requestDataPP     `json:"pp" bson:"pp"`
	Country    string            `json:"country" bson:"country"`
	Level      float32           `json:"level" bson:"level"`
	Accuracy   float32           `json:"accuracy" bson:"accuracy"`
}

type requestDataCounts struct {
	Count50s  int `json:"50" bson:"50"`
	Count100s int `json:"100" bson:"100"`
	Count300s int `json:"300" bson:"300"`
	SS        int `json:"SS" bson:"SS"`
	S         int `json:"S" bson:"S"`
	A         int `json:"A" bson:"A"`
	Plays     int `json:"plays" bson:"plays"`
}

type requestDataScores struct {
	Ranked int `json:"ranked" bson:"ranked"`
	Total  int `json:"total" bson:"total"`
}

type requestDataPP struct {
	Raw         float32 `json:"raw" bson:"raw"`
	Rank        int     `json:"rank" bson:"rank"`
	CountryRank int     `json:"countryRank" bson:"countryRank"`
}
