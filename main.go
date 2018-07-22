package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/globalsign/mgo/bson"
	"golang.org/x/text/language"

	"github.com/go-bongo/bongo"

	"github.com/mrjones/oauth"

	ctxt "context"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/op/go-logging"
	redistore "gopkg.in/boj/redistore.v1"
)

var log = logging.MustGetLogger("prosu")

/* Set up session store */
var sessionStore *redistore.RediStore

/* Set up templates */
var templates = template.Must(template.ParseGlob("templates/*"))

/* Twitter Consumer Key and Secret */
var consumerKey string
var consumerSecret string
var twitterConsumer *oauth.Consumer

// Domain
var domain string

type sessionTokenStorer struct {
	Token *oauth.RequestToken
}

// Database variables
var mongoConnectionString string
var mongoDatabase string
var connection *bongo.Connection

// i18n bundle
var bundle *i18n.Bundle

func init() {
	/* First, setting up logging */
	loggingBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	loggingBackendFormatter := logging.NewBackendFormatter(loggingBackend, format)
	logging.SetBackend(loggingBackendFormatter)
	/* Second, as long as we aren't in the production environment, try to load a .env for configuration */
	if os.Getenv("ENVIRONMENT") != "production" {
		if err := godotenv.Load(); err != nil {
			log.Warning("Couldn't load .env file")
			fmt.Println(err)
		} else {
			log.Info("Successfully loaded .env file")
		}
	} else {
		log.Debug("Production environment, not loading .env")
	}

	// Initialize sessionStore
	sessionStore = setupSessionStore()

	// Initialize consumer key and secret
	consumerKey = os.Getenv("CONSUMER_KEY")
	consumerSecret = os.Getenv("CONSUMER_SECRET")

	// Initialize domain
	domain = os.Getenv("DOMAIN")

	// Initialize Twitter client
	twitterConsumer = oauth.NewConsumer(
		consumerKey,
		consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authenticate",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		},
	)

	// Add sessionTokenStorer to gob
	gob.Register(sessionTokenStorer{})

	// Connect to MongoDB
	mongoConnectionString = os.Getenv("MONGO_URL")
	conn, err := bongo.Connect(&bongo.Config{
		ConnectionString: mongoConnectionString,
	})
	if err != nil {
		panic(err)
	}

	connection = conn

}

func main() {
	/* Set up chi router */
	// Initialize the router
	r := chi.NewRouter()

	// Initialize the middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(getLoggedInValue)

	r.Get("/", homePage)
	r.Get("/connect/twitter", redirectToTwitter)
	r.Get("/connect/twitter/callback", obtainAccessToken)
	r.Get("/logout", logoutUser)
	//FileServer(r, "/assets", http.Dir("./static"))

	r.Get("/favicon.ico", ServeFavicon)

	// Set up translator
	bundle = &i18n.Bundle{DefaultLanguage: language.English}
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	bundle.MustLoadMessageFile("./translations/active.en.toml")

	/* Listen */
	port := "5000"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	defer connection.Session.Close()
	http.ListenAndServe(":"+port, context.ClearHandler(r))
}

func getLoggedInValue(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, sessionError := sessionStore.Get(r, "prosu_session")
		ctx := r.Context()
		if sessionError != nil {
			log.Error("Error getting session")
			ctx = ctxt.WithValue(ctx, "session_error", sessionError.Error())
		} else {
			isAuthenticated := false
			if session.Values["isAuthenticated"] == true {
				isAuthenticated = true
			}
			ctx = ctxt.WithValue(ctx, "session_error", "")
			ctx = ctxt.WithValue(ctx, "session", session)
			ctx = ctxt.WithValue(ctx, "isAuthenticated", isAuthenticated)
			if isAuthenticated {
				user := &User{}
				err := connection.Collection("usermodels").FindById(bson.ObjectIdHex(session.Values["user_id"].(string)), user)
				if err != nil {
					ctx = ctxt.WithValue(ctx, "user_error", err.Error())
				} else {
					ctx = ctxt.WithValue(ctx, "user_error", "")
					ctx = ctxt.WithValue(ctx, "user", user)
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func logoutUser(w http.ResponseWriter, r *http.Request) {
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
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Error("There was an error saving the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error saving user session\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

type navbarTranslations struct {
	SignIn string
	Logout string
}

func translateNavbar(localizer *i18n.Localizer, isAuthenticated bool, user User) navbarTranslations {
	signIn := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "NavbarSignIn",
	})

	var username string
	if isAuthenticated {
		username = user.Twitter.Profile.Handle
	}

	logout := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "NavbarLogout",
		TemplateData: map[string]string{
			"Handle": username,
		},
	})
	return navbarTranslations{
		SignIn: signIn,
		Logout: logout,
	}
}
