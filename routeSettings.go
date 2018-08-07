package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-bongo/bongo"

	"github.com/globalsign/mgo/bson"
	osuapi "github.com/wcalandro/osuapi-go"

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
	ErrorFlash      []interface{}
	SuccessFlash    []interface{}
	Frequencies     [3]string
	Hours           [24]string
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
	HourToPostLabel            string
	PostFrequencyLabel         string
	PostFrequencyDaily         string
	PostFrequencyWeekly        string
	PostFrequencyMonthly       string
	CurrentUTCTimeLabel        string
}

var allOsuModes = [4]string{"osu!standard", "osu!taiko", "osu!catch", "osu!mania"}
var hours = [24]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23"}

func routeSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
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
			routeError(w, "Error getting osu! player information from database", err, middleware.GetReqID(ctx), 500)
			return
		}
	}

	errorFlashes := session.Flashes("settings_error")
	successFlashes := session.Flashes("settings_success")
	session.Save(r, w)
	pageData := settingsPageData{
		User:            user,
		OsuPlayer:       player,
		IsAuthenticated: true,
		Translations:    translations,
		Modes:           allOsuModes,
		ErrorFlash:      errorFlashes,
		SuccessFlash:    successFlashes,
		Frequencies:     [3]string{translations.PostFrequencyDaily, translations.PostFrequencyWeekly, translations.PostFrequencyMonthly},
		Hours:           hours,
	}

	templates.ExecuteTemplate(w, "settings.html", pageData)
}

func enableTweetPosting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)
	if user.OsuSettings.Enabled {
		http.Redirect(w, r, "/settings", 302)
		return
	}

	user.OsuSettings.Enabled = true
	log.Debug("User " + user.Twitter.Profile.Handle + " just enabled Tweet posting!")
	err := connection.Collection("usermodels").Save(&user)
	if err != nil {
		routeError(w, "Error saving user when enabling tweets", err, middleware.GetReqID(ctx), 500)
		return
	}
	http.Redirect(w, r, "/settings", 302)
}

func disableTweetPosting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)
	if user.OsuSettings.Enabled == false {
		http.Redirect(w, r, "/settings", 302)
		return
	}

	user.OsuSettings.Enabled = false
	log.Debug("User " + user.Twitter.Profile.Handle + " just disabled Tweet posting!")
	err := connection.Collection("usermodels").Save(&user)
	if err != nil {
		routeError(w, "Error saving user when disabling tweets", err, middleware.GetReqID(ctx), 500)
		return
	}
	http.Redirect(w, r, "/settings", 302)
}

func updateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user session", errors.New(sessionError), reqID, 500)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// Privileged page. If the user isn't authenticated, we need to redirect the user to login
	if isAuthenticated == false {
		http.Redirect(w, r, "/connect/twitter", 302)
		return
	}

	var user User
	userError := ctx.Value("user_error").(string)
	if userError != "" {
		log.Error("There was an error getting the user's account info")
		log.Error(userError)
		reqID := middleware.GetReqID(ctx)
		routeError(w, "Error getting user account info", errors.New(userError), reqID, 500)
		return
	}
	user = *ctx.Value("user").(*User)
	log.Debug("User " + user.Twitter.Profile.Handle + " is trying to update their settings....")
	if user.OsuSettings.Enabled == false {
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Parse the form
	err := r.ParseForm()
	if err != nil {
		captureError(err)
		session.AddFlash("Error parsing form", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Check to see if the player name field was filled out
	playerName := r.Form.Get("osu_username")
	if len(playerName) == 0 {
		session.AddFlash("The player name cannot be empty", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Check that the mode is valid
	modeNumber, err := strconv.Atoi(r.Form.Get("game_mode"))
	if err != nil {
		session.AddFlash("Invalid mode", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	if modeNumber < 0 || modeNumber > 3 {
		session.AddFlash("Invalid mode", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	user.OsuSettings.Mode = modeNumber

	// Check the hour to post field is valid
	hourToPostValue, err := strconv.Atoi(r.Form.Get("hour_to_post"))
	if err != nil {
		session.AddFlash("Invalid hour", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	if hourToPostValue < 0 || hourToPostValue > 23 {
		session.AddFlash("Invalid hour", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	user.OsuSettings.HourToPost = hourToPostValue

	// Check the post frequency field is valid
	postFrequencyValue, err := strconv.Atoi(r.Form.Get("post_frequency"))
	if err != nil {
		session.AddFlash("Invalid frequency", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	if postFrequencyValue < 0 || postFrequencyValue > 2 {
		session.AddFlash("Invalid frequency", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	user.OsuSettings.PostFrequency = postFrequencyValue

	// Get osu! player information
	osuPlayer, err := api.GetUser(osuapi.M{"u": playerName, "m": strconv.Itoa(modeNumber)})

	if err != nil {
		captureError(err)
		session.AddFlash("Error getting user information", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}

	// Check if the user already exists in our database
	dbOsuPlayer := &OsuPlayer{}
	err = connection.Collection("osuplayermodels").FindOne(bson.M{"userid": osuPlayer.UserID}, dbOsuPlayer)

	// We got some kind of error, either a database error or the player couldn't be found in our database
	if err != nil {
		if _, ok := err.(*bongo.DocumentNotFoundError); ok {
			log.Debug("User " + user.Twitter.Profile.Handle + " osu!player " + playerName + "(" + osuPlayer.UserID + ") doesn't exist in the database, adding...")
			// Player isn't in the database yet.
			dbOsuPlayer.UserID = osuPlayer.UserID
			dbOsuPlayer.PlayerName = osuPlayer.Username
			dbOsuPlayer.LastChecked = time.Now().Unix()
			dbOsuPlayer.Modes = OsuModes{
				Standard: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
				Mania: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
				Taiko: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
				CTB: OsuModeChecks{
					Checks: []bson.ObjectId{},
				},
			}
			err := connection.Collection("osuplayermodels").Save(dbOsuPlayer)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving new osu! player", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			osuRequest := &OsuRequest{
				OsuPlayer:   dbOsuPlayer.GetId(),
				DateChecked: time.Now().Unix(),
				Data: OsuRequestData{
					PlayerID:   osuPlayer.UserID,
					PlayerName: osuPlayer.Username,
					Counts: requestDataCounts{
						Count50s:  osuPlayer.Count50,
						Count100s: osuPlayer.Count100,
						Count300s: osuPlayer.Count300,
						SS:        osuPlayer.CountRankSS + osuPlayer.CountRankSSH,
						S:         osuPlayer.CountRankS + osuPlayer.CountRankSH,
						A:         osuPlayer.CountRankA,
						Plays:     osuPlayer.Playcount,
					},
					Scores: requestDataScores{
						Ranked: osuPlayer.RankedScore,
						Total:  osuPlayer.TotalScore,
					},
					PP: requestDataPP{
						Raw:         osuPlayer.PP,
						Rank:        osuPlayer.GlobalRank,
						CountryRank: osuPlayer.CountryRank,
					},
					Country:  osuPlayer.Country,
					Level:    osuPlayer.Level,
					Accuracy: osuPlayer.Accuracy,
				},
			}

			err = connection.Collection("osurequestmodels").Save(osuRequest)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving osu! data request to database", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			if modeNumber == 0 {
				// osu! standard
				dbOsuPlayer.Modes.Standard.Checks = append(dbOsuPlayer.Modes.Standard.Checks, osuRequest.GetId())
			} else if modeNumber == 1 {
				// osu! taiko
				dbOsuPlayer.Modes.Taiko.Checks = append(dbOsuPlayer.Modes.Taiko.Checks, osuRequest.GetId())
			} else if modeNumber == 2 {
				// osu! catch
				dbOsuPlayer.Modes.CTB.Checks = append(dbOsuPlayer.Modes.CTB.Checks, osuRequest.GetId())
			} else if modeNumber == 3 {
				// osu! mania
				dbOsuPlayer.Modes.Mania.Checks = append(dbOsuPlayer.Modes.Mania.Checks, osuRequest.GetId())
			}
			err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
			if err != nil {
				captureError(err)
				session.AddFlash("Error re-saving osu! player to database", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			user.OsuSettings.Player = dbOsuPlayer.GetId()
			user.OsuSettings.Mode = modeNumber
			err = connection.Collection("usermodels").Save(&user)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving final settings", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			log.Debug("User " + user.Twitter.Profile.Handle + " successfully added user to database and changed their mode to " + allOsuModes[modeNumber])
			session.AddFlash("Successfully updated settings", "settings_success")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		captureError(err)
		session.AddFlash("Error checking if the user already exists in the database", "settings_error")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
	// The player could be found in the database, now we have to check if they have a recent osu! API request saved to their user for their selected game mode. If so, we don't want to save more data.
	log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " exists in the database, checking to see if we have recent data for mode " + strconv.Itoa(modeNumber))

	// Check to see if the name has changed, if so, update it
	if osuPlayer.Username != strings.ToLower(dbOsuPlayer.PlayerName) {
		log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + "'s name has changed. Updating")
		dbOsuPlayer.PlayerName = osuPlayer.Username
		err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
		if err != nil {
			captureError(err)
			session.AddFlash("Error updating player name in database", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
	}

	if modeNumber == 0 {
		// Standard

		// First check if they have any requests for this game mode at all
		if len(dbOsuPlayer.Modes.Standard.Checks) != 0 {
			log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " has data for mode " + allOsuModes[modeNumber] + ". Saving settings and returning")
			user.OsuSettings.Player = dbOsuPlayer.GetId()
			err = connection.Collection("usermodels").Save(&user)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving user after updating player info and mode, while not grabbing new data for osu!standard", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
			session.AddFlash("Successfully updated settings", "settings_success")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " doesn't have data for mode " + allOsuModes[modeNumber] + ". Saving data, settings, and returning")
		// They don't have any recent checks saved, we have to save one
		osuRequest := &OsuRequest{
			OsuPlayer:   dbOsuPlayer.GetId(),
			DateChecked: time.Now().Unix(),
			Data: OsuRequestData{
				PlayerID:   osuPlayer.UserID,
				PlayerName: osuPlayer.Username,
				Counts: requestDataCounts{
					Count50s:  osuPlayer.Count50,
					Count100s: osuPlayer.Count100,
					Count300s: osuPlayer.Count300,
					SS:        osuPlayer.CountRankSS + osuPlayer.CountRankSSH,
					S:         osuPlayer.CountRankS + osuPlayer.CountRankSH,
					A:         osuPlayer.CountRankA,
					Plays:     osuPlayer.Playcount,
				},
				Scores: requestDataScores{
					Ranked: osuPlayer.RankedScore,
					Total:  osuPlayer.TotalScore,
				},
				PP: requestDataPP{
					Raw:         osuPlayer.PP,
					Rank:        osuPlayer.GlobalRank,
					CountryRank: osuPlayer.CountryRank,
				},
				Country:  osuPlayer.Country,
				Level:    osuPlayer.Level,
				Accuracy: osuPlayer.Accuracy,
			},
		}
		err = connection.Collection("osurequestmodels").Save(osuRequest)
		if err != nil {
			captureError(err)
			session.AddFlash("Error saving new data entry for osu!standard", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		dbOsuPlayer.Modes.Standard.Checks = append(dbOsuPlayer.Modes.Standard.Checks, osuRequest.GetId())
		err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
		if err != nil {
			captureError(err)
			session.AddFlash("Error osu! player with updated data", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		user.OsuSettings.Player = dbOsuPlayer.GetId()
		err = connection.Collection("usermodels").Save(&user)
		if err != nil {
			captureError(err)
			session.AddFlash("Error updating information for user", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
		session.AddFlash("Successfully updated settings", "settings_success")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	} else if modeNumber == 1 {
		// Taiko
		// First check if they have any requests for this game mode at all
		if len(dbOsuPlayer.Modes.Taiko.Checks) != 0 {
			log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " has data for mode " + allOsuModes[modeNumber] + ". Saving settings and returning")
			user.OsuSettings.Player = dbOsuPlayer.GetId()
			err = connection.Collection("usermodels").Save(&user)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving user after updating player info and mode, while not grabbing new data for osu!taiko", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
			session.AddFlash("Successfully updated settings", "settings_success")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " doesn't have recent data for mode " + strconv.Itoa(modeNumber) + ". Saving data, settings, and returning")
		// They don't have any recent checks saved, we have to save one
		osuRequest := &OsuRequest{
			OsuPlayer:   dbOsuPlayer.GetId(),
			DateChecked: time.Now().Unix(),
			Data: OsuRequestData{
				PlayerID:   osuPlayer.UserID,
				PlayerName: osuPlayer.Username,
				Counts: requestDataCounts{
					Count50s:  osuPlayer.Count50,
					Count100s: osuPlayer.Count100,
					Count300s: osuPlayer.Count300,
					SS:        osuPlayer.CountRankSS + osuPlayer.CountRankSSH,
					S:         osuPlayer.CountRankS + osuPlayer.CountRankSH,
					A:         osuPlayer.CountRankA,
					Plays:     osuPlayer.Playcount,
				},
				Scores: requestDataScores{
					Ranked: osuPlayer.RankedScore,
					Total:  osuPlayer.TotalScore,
				},
				PP: requestDataPP{
					Raw:         osuPlayer.PP,
					Rank:        osuPlayer.GlobalRank,
					CountryRank: osuPlayer.CountryRank,
				},
				Country:  osuPlayer.Country,
				Level:    osuPlayer.Level,
				Accuracy: osuPlayer.Accuracy,
			},
		}
		err = connection.Collection("osurequestmodels").Save(osuRequest)
		if err != nil {
			captureError(err)
			session.AddFlash("Error saving new data entry for osu!taiko", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		dbOsuPlayer.Modes.Taiko.Checks = append(dbOsuPlayer.Modes.Taiko.Checks, osuRequest.GetId())
		err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
		if err != nil {
			captureError(err)
			session.AddFlash("Error osu! player with updated data", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		user.OsuSettings.Player = dbOsuPlayer.GetId()
		err = connection.Collection("usermodels").Save(&user)
		if err != nil {
			captureError(err)
			session.AddFlash("Error updating information for user", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
		session.AddFlash("Successfully updated settings", "settings_success")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	} else if modeNumber == 2 {
		// CTB
		// First check if they have any requests for this game mode at all
		if len(dbOsuPlayer.Modes.CTB.Checks) != 0 {
			log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " has data for mode " + allOsuModes[modeNumber] + ". Saving settings and returning")
			user.OsuSettings.Player = dbOsuPlayer.GetId()
			err = connection.Collection("usermodels").Save(&user)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving user after updating player info and mode, while not grabbing new data for osu!catch", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
			session.AddFlash("Successfully updated settings", "settings_success")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " doesn't have data for mode " + allOsuModes[modeNumber] + ". Saving data, settings, and returning")
		// They don't have any recent checks saved, we have to save one
		osuRequest := &OsuRequest{
			OsuPlayer:   dbOsuPlayer.GetId(),
			DateChecked: time.Now().Unix(),
			Data: OsuRequestData{
				PlayerID:   osuPlayer.UserID,
				PlayerName: osuPlayer.Username,
				Counts: requestDataCounts{
					Count50s:  osuPlayer.Count50,
					Count100s: osuPlayer.Count100,
					Count300s: osuPlayer.Count300,
					SS:        osuPlayer.CountRankSS + osuPlayer.CountRankSSH,
					S:         osuPlayer.CountRankS + osuPlayer.CountRankSH,
					A:         osuPlayer.CountRankA,
					Plays:     osuPlayer.Playcount,
				},
				Scores: requestDataScores{
					Ranked: osuPlayer.RankedScore,
					Total:  osuPlayer.TotalScore,
				},
				PP: requestDataPP{
					Raw:         osuPlayer.PP,
					Rank:        osuPlayer.GlobalRank,
					CountryRank: osuPlayer.CountryRank,
				},
				Country:  osuPlayer.Country,
				Level:    osuPlayer.Level,
				Accuracy: osuPlayer.Accuracy,
			},
		}
		err = connection.Collection("osurequestmodels").Save(osuRequest)
		if err != nil {
			captureError(err)
			session.AddFlash("Error saving new data entry for osu!catch", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		dbOsuPlayer.Modes.CTB.Checks = append(dbOsuPlayer.Modes.CTB.Checks, osuRequest.GetId())
		err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
		if err != nil {
			captureError(err)
			session.AddFlash("Error osu! player with updated data", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		user.OsuSettings.Player = dbOsuPlayer.GetId()
		err = connection.Collection("usermodels").Save(&user)
		if err != nil {
			captureError(err)
			session.AddFlash("Error updating information for user", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
		session.AddFlash("Successfully updated settings", "settings_success")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	} else if modeNumber == 3 {
		// Mania
		// First check if they have any requests for this game mode at all
		if len(dbOsuPlayer.Modes.Mania.Checks) != 0 {
			log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " has data for mode " + allOsuModes[modeNumber] + ". Saving settings and returning")
			user.OsuSettings.Player = dbOsuPlayer.GetId()
			err = connection.Collection("usermodels").Save(&user)
			if err != nil {
				captureError(err)
				session.AddFlash("Error saving user after updating player info and mode, while not grabbing new data for osu!mania", "settings_error")
				session.Save(r, w)
				http.Redirect(w, r, "/settings", 302)
				return
			}
			log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
			session.AddFlash("Successfully updated settings", "settings_success")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s player " + dbOsuPlayer.PlayerName + " doesn't have data for mode " + allOsuModes[modeNumber] + ". Saving data, settings, and returning")
		// They don't have any recent checks saved, we have to save one
		osuRequest := &OsuRequest{
			OsuPlayer:   dbOsuPlayer.GetId(),
			DateChecked: time.Now().Unix(),
			Data: OsuRequestData{
				PlayerID:   osuPlayer.UserID,
				PlayerName: osuPlayer.Username,
				Counts: requestDataCounts{
					Count50s:  osuPlayer.Count50,
					Count100s: osuPlayer.Count100,
					Count300s: osuPlayer.Count300,
					SS:        osuPlayer.CountRankSS + osuPlayer.CountRankSSH,
					S:         osuPlayer.CountRankS + osuPlayer.CountRankSH,
					A:         osuPlayer.CountRankA,
					Plays:     osuPlayer.Playcount,
				},
				Scores: requestDataScores{
					Ranked: osuPlayer.RankedScore,
					Total:  osuPlayer.TotalScore,
				},
				PP: requestDataPP{
					Raw:         osuPlayer.PP,
					Rank:        osuPlayer.GlobalRank,
					CountryRank: osuPlayer.CountryRank,
				},
				Country:  osuPlayer.Country,
				Level:    osuPlayer.Level,
				Accuracy: osuPlayer.Accuracy,
			},
		}
		err = connection.Collection("osurequestmodels").Save(osuRequest)
		if err != nil {
			captureError(err)
			session.AddFlash("Error saving new data entry for osu!mania", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		dbOsuPlayer.Modes.Mania.Checks = append(dbOsuPlayer.Modes.Mania.Checks, osuRequest.GetId())
		err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
		if err != nil {
			captureError(err)
			session.AddFlash("Error osu! player with updated data", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		user.OsuSettings.Player = dbOsuPlayer.GetId()
		err = connection.Collection("usermodels").Save(&user)
		if err != nil {
			captureError(err)
			session.AddFlash("Error updating information for user", "settings_error")
			session.Save(r, w)
			http.Redirect(w, r, "/settings", 302)
			return
		}
		log.Debug("User " + user.Twitter.Profile.Handle + "'s settings are now updated, returning")
		session.AddFlash("Successfully updated settings", "settings_success")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", 302)
		return
	}
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

	hourToPostLabel := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsHourToPostLabel",
	})

	postFrequencyLabel := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPostFrequencyLabel",
	})

	postFrequencyDaily := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPostFrequencyDaily",
	})

	postFrequencyWeekly := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPostFrequencyWeekly",
	})

	postFrequencyMonthly := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsPostFrequencyMonthly",
	})

	currentUTCTimeLabel := localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "SettingsCurrentUTCTimeLabel",
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
		HourToPostLabel:            hourToPostLabel,
		PostFrequencyLabel:         postFrequencyLabel,
		PostFrequencyDaily:         postFrequencyDaily,
		PostFrequencyWeekly:        postFrequencyWeekly,
		PostFrequencyMonthly:       postFrequencyMonthly,
		CurrentUTCTimeLabel:        currentUTCTimeLabel,
	}
}
