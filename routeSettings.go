package main

import (
	"errors"
	"net/http"

	"github.com/globalsign/mgo/bson"

	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type settingsPageData struct {
	User            User
	Translations    settingsPageTranslations
	IsAuthenticated bool
	OsuPlayer       OsuPlayer
	Modes           [4]string
}

type settingsPageTranslations struct {
	Navbar                     navbarTranslations
	SettingsHeader             string
	TweetPostingText           string
	TweetPostingStatusEnabled  string
	TweetPostingStatusDisabled string
	EnableTweetPosting         string
	DisableTweetPosting        string
	OsuUsernameText            string
	OsuUsernamePlaceholder     string
	GameModeText               string
	UpdateSettingsButton       string
	NoDataWarning              string
}

func routeSettings(w http.ResponseWriter, r *http.Request) {
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

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", http.StatusTemporaryRedirect)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, http.StatusInternalServerError)
		return
	}
	user = *ctx.Value("user").(*User)

	// Localization
	lang := session.Values["language"].(string)
	accept := r.Header.Get("Accept-Language")
	localizer := i18n.NewLocalizer(bundle, lang, accept)

	translations := translateSettingsPage(localizer, isAuthenticated, user)

	// Grab the osu! player in the user's settings if it is currently set
	player := OsuPlayer{}
	if bson.IsObjectIdHex(user.OsuSettings.Player.Hex()) {
		err := connection.Collection("osuplayermodels").FindById(bson.ObjectIdHex(user.OsuSettings.Player.Hex()), &player)
		if err != nil {
			routeError(w, "Error getting osu! player information from database", err, middleware.GetReqID(ctx), http.StatusInternalServerError)
			return
		}
	}

	modes := [4]string{"osu!standard", "osu!taiko", "osu!catch", "osu!mania"}
	pageData := settingsPageData{
		User:            user,
		OsuPlayer:       player,
		IsAuthenticated: true,
		Translations:    translations,
		Modes:           modes,
	}

	//if(user.OsuSettings.Player != nil)
	templates.ExecuteTemplate(w, "settings.html", pageData)
}

func translateSettingsPage(localizer *i18n.Localizer, isAuthenticated bool, user User) settingsPageTranslations {
	navbar := translateNavbar(localizer, isAuthenticated, user)

	settingsHeaderText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageSettingsHeader",
	})

	settingsTweetPostingText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageTweetPostingText",
	})

	settingsTweetPostingEnabled := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageTweetPostingStatusEnabled",
	})

	settingsTweetPostingDisabled := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageTweetPostingStatusDisabled",
	})

	settingsOsuUsernameText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageOsuUsernameText",
	})

	settingsOsuUsernamePlaceholder := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageOsuUsernamePlaceholder",
	})

	settingsEnableTweetPosting := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageEnableTweetPostingButton",
	})

	settingsDisableTweetPosting := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageDisableTweetPostingButton",
	})

	gameModeText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageGameModeText",
	})

	updateSettings := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageUpdateSettingsButton",
	})

	noDataWarning := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPageNoDataWarning",
	})

	return settingsPageTranslations{
		Navbar:                     navbar,
		SettingsHeader:             settingsHeaderText,
		TweetPostingText:           settingsTweetPostingText,
		TweetPostingStatusEnabled:  settingsTweetPostingEnabled,
		TweetPostingStatusDisabled: settingsTweetPostingDisabled,
		EnableTweetPosting:         settingsEnableTweetPosting,
		DisableTweetPosting:        settingsDisableTweetPosting,
		OsuUsernameText:            settingsOsuUsernameText,
		OsuUsernamePlaceholder:     settingsOsuUsernamePlaceholder,
		GameModeText:               gameModeText,
		UpdateSettingsButton:       updateSettings,
		NoDataWarning:              noDataWarning,
	}
}

func enableTweetPosting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, http.StatusInternalServerError)
		return
	}
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", http.StatusTemporaryRedirect)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, http.StatusInternalServerError)
		return
	}
	user = *ctx.Value("user").(*User)
	if user.OsuSettings.Enabled {
		http.Redirect(w, r, "/settings", http.StatusTemporaryRedirect)
		return
	}

	user.OsuSettings.Enabled = true
	err := connection.Collection("usermodels").Save(&user)
	if err != nil {
		routeError(w, "Error saving user when enabling tweets", err, middleware.GetReqID(ctx), 500)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusTemporaryRedirect)
}
