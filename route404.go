package main

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type notFoundPageData struct {
	Session         *sessions.Session
	IsAuthenticated bool
	User            User
	Translations    notFoundPageTranslations
}

type notFoundPageTranslations struct {
	Navbar       navbarTranslations
	NotFoundText string
}

func notFound(w http.ResponseWriter, r *http.Request) {
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

	// Localization
	lang := session.Values["language"].(string)
	accept := r.Header.Get("Accept-Language")
	localizer := i18n.NewLocalizer(bundle, lang, accept)

	translations := translateNotFoundPage(localizer, isAuthenticated, user)

	pageData := notFoundPageData{
		Session:         session,
		IsAuthenticated: isAuthenticated,
		User:            user,
		Translations:    translations,
	}

	w.WriteHeader(http.StatusNotFound)
	templates.ExecuteTemplate(w, "404.html", pageData)
}

func translateNotFoundPage(localizer *i18n.Localizer, isAuthenticated bool, user User) notFoundPageTranslations {
	navbar := translateNavbar(localizer, isAuthenticated, user)
	notFoundText := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "NotFoundText",
	})

	return notFoundPageTranslations{
		Navbar:       navbar,
		NotFoundText: notFoundText,
	}
}
