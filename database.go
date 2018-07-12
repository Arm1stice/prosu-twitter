package main

import (
	"github.com/globalsign/mgo/bson"
	"github.com/go-bongo/bongo"
)

/* usermodels */

// User - A user in prosu
type User struct {
	bongo.DocumentBase `bson:",inline"`
	OsuSettings        OsuSettings `bson:"osuSettings"`
	TweetHistory       []UserTweet `bson:"tweetHistory"`
	Twitter            TwitterUser `bson:"twitter"`
}

// OsuSettings - The osu-related settings for a user in Prosu
type OsuSettings struct {
	Player  bson.ObjectId `bson:"player"`
	Mode    int           `bson:"mode"`
	Enabled bool          `bson:"enabled"`
}

// UserTweet - A tweet object
type UserTweet struct {
	DatePosted  int         `bson:"datePosted"`
	TweetObject TweetObject `bson:"tweetObject"`
}

// TweetObject - The object inside the UserTweet containing the tweet ID
type TweetObject struct {
	ID string `bson:"id"`
}

// TwitterUser - Contains twitter user information
type TwitterUser struct {
	Token       string         `bson:"token"`
	TokenSecret string         `bson:"tokenSecret"`
	Profile     TwitterProfile `bson:"profile"`
}

// TwitterProfile - Contains profile information for a TwitterUser
type TwitterProfile struct {
	Username    string `bson:"username"`
	DisplayName string `bson:"displayName"`
	ID          string `bson:"string"`
}
