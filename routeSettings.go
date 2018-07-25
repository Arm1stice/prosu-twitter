package main

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type settingsPageData struct {
	User            User
	Translations    settingsPageTranslations
	IsAuthenticated bool
}

type settingsPageTranslations struct {
	Navbar                     navbarTranslations
	SettingsHeader             string
	TweetPostingText           string
	TweetPostingStatusEnabled  string
	TweetPostingStatusDisabled string
}

func routeSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", reqID, http.StatusInternalServerError)
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
		routeError(w, "Error getting user account info", reqID, http.StatusInternalServerError)
		return
	}
	user = *ctx.Value("user").(*User)

	// Localization
	lang := session.Values["language"].(string)
	accept := r.Header.Get("Accept-Language")
	localizer := i18n.NewLocalizer(bundle, lang, accept)

	translations := translateSettingsPage(localizer, isAuthenticated, user)

	pageData := settingsPageData{
		User:            user,
		IsAuthenticated: true,
		Translations:    translations,
	}

	// TODO: Create settings.html
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

	return settingsPageTranslations{
		Navbar:                     navbar,
		SettingsHeader:             settingsHeaderText,
		TweetPostingText:           settingsTweetPostingText,
		TweetPostingStatusEnabled:  settingsTweetPostingEnabled,
		TweetPostingStatusDisabled: settingsTweetPostingDisabled,
	}
}
