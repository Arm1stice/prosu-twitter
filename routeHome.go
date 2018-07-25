package main

import (
	"errors"
	"net/http"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/middleware"
)

/* Home page data structs */
type homePageData struct {
	Session         *sessions.Session
	IsAuthenticated bool
	User            User
	Translations    homePageTranslations
	TotalUsers      string
	TotalTweets     string
}

type homePageTranslations struct {
	Navbar             navbarTranslations
	TotalUsers         string
	TotalTweets        string
	SettingsButtonText string
}

var currentUsers string
var totalTweets string

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
	currentUsers = humanize.Comma(int64(count))
}

func updateTotalTweets() {
	total := 0
	user := &User{}
	rSet := connection.Collection("usermodels").Find(bson.M{})

	for rSet.Next(user) {
		total = total + len(user.TweetHistory)
	}
	totalTweets = humanize.Comma(int64(total))
}

// When someone visits the home page
func homePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, http.StatusInternalServerError)
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
			routeError(w, "Error getting user account info", errors.New(userError), reqID, http.StatusInternalServerError)
			return
		}
		user = *ctx.Value("user").(*User)
	}

	// Localization
	lang := session.Values["language"].(string)
	accept := r.Header.Get("Accept-Language")
	localizer := i18n.NewLocalizer(bundle, lang, accept)

	translations := translateHomePage(localizer, isAuthenticated, user)

	pageData := homePageData{
		Session:         session,
		IsAuthenticated: isAuthenticated,
		User:            user,
		Translations:    translations,
		TotalUsers:      currentUsers,
		TotalTweets:     totalTweets,
	}

	templates.ExecuteTemplate(w, "index.html", pageData)
}

func translateHomePage(localizer *i18n.Localizer, isAuthenticated bool, user User) homePageTranslations {
	navbar := translateNavbar(localizer, isAuthenticated, user)
	homePageTotalUsers := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "HomePageTotalUsers",
		TemplateData: map[string]string{
			"TotalUsers": currentUsers,
		},
	})
	homePageTotalTweets := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "HomePageTotalTweets",
		TemplateData: map[string]string{
			"TotalTweets": totalTweets,
		},
	})

	homePageSettingsButton := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "HomePageSettingsButton",
	})

	return homePageTranslations{
		Navbar:             navbar,
		TotalUsers:         homePageTotalUsers,
		TotalTweets:        homePageTotalTweets,
		SettingsButtonText: homePageSettingsButton,
	}
}
