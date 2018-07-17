package main

import (
	"net/http"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/sessions"

	"github.com/go-chi/chi/middleware"
)

/* Home page interface */
type homeInterface struct {
	Session         *sessions.Session
	IsAuthenticated bool
	User            User
	CurrentUsers    int
	TotalTweets     int
}

var currentUsers = 0
var totalTweets = 0

func init() {
	setInterval(updateCurrentUsers, 60*1000, true)
	setInterval(updateTotalTweets, 60*60*1000, true)
	setTimeout(updateCurrentUsers, 5000)
	setTimeout(updateTotalTweets, 5000)
}

func updateCurrentUsers() {
	rSet := connection.Collection("usermodels").Find(bson.M{})
	count, err := CountResults(rSet)
	if err != nil {
		log.Error("Error updating current users")
		log.Error(err.Error())
		return
	}
	currentUsers = count
}

func updateTotalTweets() {
	total := 0
	user := &User{}
	rSet := connection.Collection("usermodels").Find(bson.M{})

	for rSet.Next(user) {
		total = total + len(user.TweetHistory)
	}
	totalTweets = total
}

// When someone visits the home page
func homePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error getting user session\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	isAuthenticated := ctx.Value("isAuthenticated").(bool)
	var user User
	if isAuthenticated {
		userError := ctx.Value("user_error").(string)
		if userError != "" {
			log.Error("There was an error getting the user's account info")
			log.Error(userError)
			reqID := middleware.GetReqID(ctx)
			http.Error(w, "Error getting user account info\nRequestID: "+reqID, http.StatusInternalServerError)
			return
		}
		user = *ctx.Value("user").(*User)
	}
	pageData := homeInterface{
		Session:         session,
		IsAuthenticated: isAuthenticated,
		User:            user,
		CurrentUsers:    currentUsers,
		TotalTweets:     totalTweets,
	}
	templates.ExecuteTemplate(w, "index.html", pageData)
}
