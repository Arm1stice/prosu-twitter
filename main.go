package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/robfig/cron"
	"github.com/wcalandro/osuapi-go"

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

// osu! api
var api osuRateLimiter
var postingAPI osuRateLimiter

// Is maintenance
var isMaintenance = false

type blankData struct {
}

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

	osuAPIKey := os.Getenv("OSU_API_KEY")
	if len(osuAPIKey) == 0 {
		panic(errors.New("OSU_API_KEY variable must not be empty"))
	}
	api = newOsuLimiter(osuapi.NewAPI(osuAPIKey), 250)
	postingAPI = newOsuLimiter(osuapi.NewAPI(osuAPIKey), 250)

	// Check if maintenance mode
	if os.Getenv("MAINTENANCE") == "true" {
		isMaintenance = true
	}
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

	if isMaintenance {
		r.Use(func(next http.Handler) http.Handler {

			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				templates.ExecuteTemplate(w, "503.html", blankData{})
			})
		})
	}
	r.Use(getLoggedInValue)

	r.Get(relicHandle("/", homePage))

	r.Get(relicHandle("/connect/twitter", redirectToTwitter))
	r.Get(relicHandle("/connect/twitter/callback", obtainAccessToken))

	r.Get(relicHandle("/logout", logoutUser))

	r.Get(relicHandle("/settings", routeSettings))
	r.Post(relicHandle("/settings/enable", enableTweetPosting))
	r.Post(relicHandle("/settings/disable", disableTweetPosting))
	r.Post(relicHandle("/settings/update", updateSettings))
	//FileServer(r, "/assets", http.Dir("./static"))

	r.Get("/favicon.ico", ServeFavicon)

	// LoaderIO Configuration
	loaderIoKey := os.Getenv("LOADERIO_KEY")
	if loaderIoKey != "" {
		r.Get("/"+loaderIoKey+".txt", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, loaderIoKey)
		})
	}

	r.NotFound(notFound)

	// Set up translator
	bundle = &i18n.Bundle{DefaultLanguage: language.English}
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	bundle.MustLoadMessageFile("./translations/active.en.toml")

	// Cron job
	c := cron.New()
	c.AddFunc("0 0 * * * *", func() {
		log.Info("Running posting function")
		findAndGenerate()
	})
	c.Start()

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
			if session.Values["language"] == nil {
				session.Values["language"] = ""
			}
			session.Save(r, w)
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
		http.Error(w, "Error getting user session\nRequestID: "+reqID, 500)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Error("There was an error saving the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error saving user session\nRequestID: "+reqID, 500)
		return
	}
	http.Redirect(w, r, "/", 302)
}

type navbarTranslations struct {
	SignIn       string
	Logout       string
	HomePage     string
	SettingsPage string
}

func translateNavbar(localizer *i18n.Localizer, isAuthenticated bool, user User) navbarTranslations {
	signIn := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "NavbarSignIn",
	})

	homePage := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "NavbarHomePage",
	})

	settingsPage := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "NavbarSettingsPage",
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
		SignIn:       signIn,
		Logout:       logout,
		HomePage:     homePage,
		SettingsPage: settingsPage,
	}
}
