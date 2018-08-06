package main

import (
	"github.com/ChimeraCoder/anaconda"
	"github.com/globalsign/mgo/bson"
	"github.com/go-bongo/bongo"
	"github.com/mrjones/oauth"
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
	Player        bson.ObjectId `bson:"player,omitempty"`
	Mode          int           `bson:"mode"`
	Enabled       bool          `bson:"enabled"`
	HourToPost    int           `bson:"hourToPost"`
	PostFrequency int           `bson:"postFrequency"` // 0 = Daily, 1 = Weekly, 2 = Monthly
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
	Handle    string `bson:"username"`
	Name      string `bson:"displayName"`
	TwitterID string `bson:"id"`
}

// findOrCreateUser - Either find the user in the database or create a new one
func findOrCreateUser(twitterUser anaconda.User, accessToken *oauth.AccessToken) (User, error) {
	user := &User{}
	log.Debug("Checking whether Twitter user @" + twitterUser.ScreenName + " has an account with us.")
	err := connection.Collection("usermodels").FindOne(bson.M{"twitter.profile.id": twitterUser.IdStr}, user)
	if err != nil {
		// Check if the error was that the document wasn't found
		if _, ok := err.(*bongo.DocumentNotFoundError); ok {
			// A document wasn't found for this user, we will have to create a new user
			log.Debug("We didn't find an existing user for @" + twitterUser.ScreenName + ". We have to create one.")
			newUser := &User{
				OsuSettings: OsuSettings{
					Player:  "",
					Mode:    0,
					Enabled: false,
				},
				TweetHistory: []UserTweet{},
				Twitter: TwitterUser{
					Token:       accessToken.Token,
					TokenSecret: accessToken.Secret,
					Profile: TwitterProfile{
						Handle:    twitterUser.ScreenName,
						Name:      twitterUser.Name,
						TwitterID: twitterUser.IdStr,
					},
				},
			}
			// Failed to save user
			if err := connection.Collection("usermodels").Save(newUser); err != nil {
				log.Error("Error saving user")
				return *newUser, err
			}
			//Successfully saved user
			log.Debug("Successfully created new user for @" + twitterUser.ScreenName)
			return *newUser, nil
		}
		log.Error("An error occurred looking for the user")
		return *user, err
	}
	// We found a user in the database that matches
	log.Debug("Found an existing user for @" + twitterUser.ScreenName + ": " + user.GetId().Hex() + ". Checking to see if the handle matches.")
	if user.Twitter.Profile.Handle == twitterUser.ScreenName {
		// Handle matches
		log.Debug("Handle for @" + twitterUser.ScreenName + " matches")
		return *user, nil
	}
	// We need to update the handle in the database
	user.Twitter.Profile.Handle = twitterUser.ScreenName
	if err := connection.Collection("usermodels").Save(user); err != nil {
		// An error occurred when saving the doucument
		log.Error("Error saving user after updating handle")
		return *user, err
	}
	// Successfully updated handle in database
	return *user, nil
}
