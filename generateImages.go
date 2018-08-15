package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/dustin/go-humanize"

	"golang.org/x/image/font"

	findfont "github.com/flopp/go-findfont"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
	osuapi "github.com/wcalandro/osuapi-go"
	"gopkg.in/fogleman/gg.v1"
)

var arialFont20, arialFont12, arialFont18, arialBold20 font.Face

func gLog(msg string) {
	log.Info("[TWEET GENERATION] " + msg)
}

func gError(msg string) {
	log.Error("[TWEET GENERATION] " + msg)
}

func gDebug(msg string) {
	log.Debug("[TWEET GENERATION] " + msg)
}

func init() {
	fontPath, err := findfont.Find("arial.ttf")
	if err != nil {
		panic(err)
	}
	fontData, err := ioutil.ReadFile(fontPath)
	if err != nil {
		panic(err)
	}
	arial, err := truetype.Parse(fontData)
	if err != nil {
		panic(err)
	}
	arialFont20 = truetype.NewFace(arial, &truetype.Options{
		Size: 20,
	})
	arialFont12 = truetype.NewFace(arial, &truetype.Options{
		Size: 12,
	})
	arialFont18 = truetype.NewFace(arial, &truetype.Options{
		Size: 18,
	})
	// Bold font for player name
	fontPath, err = findfont.Find("arial bold.ttf")
	if err != nil {
		panic(err)
	}
	fontData, err = ioutil.ReadFile(fontPath)
	if err != nil {
		panic(err)
	}
	arial, err = truetype.Parse(fontData)
	if err != nil {
		panic(err)
	}
	arialBold20 = truetype.NewFace(arial, &truetype.Options{
		Size: 20,
	})
}

// Find and generate finds all users that we should be generating for and generates and posts images for them
func findAndGenerate() {
	list := []bson.ObjectId{}
	hour := time.Now().UTC().Hour()
	gLog("Time to post tweets for hour " + strconv.Itoa(hour))
	resultSet := connection.Collection("usermodels").Find(bson.M{"osuSettings.hourToPost": hour, "osuSettings.enabled": true})
	number, err := CountResults(resultSet)

	if err != nil {
		gError("Error getting number of results")
		gError(err.Error())
	}
	gLog(strconv.Itoa(number) + " preliminary results. Time to filter")
	user := &User{}
	handle := ""
	for resultSet.Next(user) {
		handle = "@" + user.Twitter.Profile.Handle
		gDebug("Checking user " + handle)
		// Check to see if they have a player set in their settings
		if user.OsuSettings.Player == "" {
			gDebug(handle + " doesn't have a valid player object in their osu! settings, skipping user.")
			continue
		}
		// Check to see if they have a recent post that doesn't fit their frequency settings
		// First check to see if they have ever had a tweet posted
		if len(user.TweetHistory) != 0 {
			lastTweet := user.TweetHistory[len(user.TweetHistory)-1]
			// If they post daily, check if they have had a tweet posted within the last 24 hours
			if user.OsuSettings.PostFrequency == 0 && time.Now().Unix()-lastTweet.DatePosted < 86400 {
				gDebug(handle + " has had a tweet posted within the last 24 hours when their account is set to post daily. Skipping...")
				continue
			}
			// If they post weekly, check if they have had a tweet posted within the last 7 days
			if user.OsuSettings.PostFrequency == 1 && time.Now().Unix()-lastTweet.DatePosted < 604800 {
				gDebug(handle + " has had a tweet posted within the last 7 days when their account is set to post weekly. Skipping...")
				continue
			}
			// If they post monthly, check if they have had a tweet posted within the last 28 days
			if user.OsuSettings.PostFrequency == 2 && time.Now().Unix()-lastTweet.DatePosted < 2419200 {
				gDebug(handle + " has had a tweet posted within the last 28 days when their account is set to post monthly. Skipping...")
				continue
			}
		}
		list = append(list, user.GetId())
	}
	gLog("Finished filtering users. We now only have " + strconv.Itoa(len(list)) + " users to post tweets for")

	// Time to generate
	for _, uID := range list {
		go updateAndPost(uID)
	}
}

func updateAndPost(userID bson.ObjectId) {
	l := pLogger{
		UserID: userID.Hex(),
	}
	// First we grab the user associated with the ObjectID we were passed
	l.Log("Grabbing Prosu user from the database")
	prosuUser := &User{}
	err := connection.Collection("usermodels").FindById(userID, prosuUser)
	if err != nil {
		l.Error("Failed to grab Prosu user from the database")
		captureError(err)
		return
	}
	l.Log("Successfully grabbed Prosu user from the database")

	// We need to run a new data check on the player that is associated with the use
	// First we grab the player from the database
	l.Log("Grabbing associated osu! player from the database")
	dbOsuPlayer := &OsuPlayer{}
	err = connection.Collection("osuplayermodels").FindById(prosuUser.OsuSettings.Player, dbOsuPlayer)
	if err != nil {
		l.Error("Failed to grab associated osu! player from the database")
		captureError(err)
		return
	}
	l.Log("Successfully grabbed associated osu! player " + dbOsuPlayer.PlayerName + " from the database")
	l.Log("Getting last check for the user's preferred game mode: " + allOsuModes[prosuUser.OsuSettings.Mode])

	// Then we need to see if one was run in the past 3 hours. This will help in case two or more people are both tracking the same person for some stupid reaosn
	lastCheck := &OsuRequest{}
	checks := []bson.ObjectId{}
	if prosuUser.OsuSettings.Mode == 0 {
		checks = dbOsuPlayer.Modes.Standard.Checks
	} else if prosuUser.OsuSettings.Mode == 1 {
		checks = dbOsuPlayer.Modes.Taiko.Checks
	} else if prosuUser.OsuSettings.Mode == 2 {
		checks = dbOsuPlayer.Modes.CTB.Checks
	} else if prosuUser.OsuSettings.Mode == 3 {
		checks = dbOsuPlayer.Modes.Mania.Checks
	}

	err = connection.Collection("osurequestmodels").FindById(checks[len(checks)-1], lastCheck)
	if err != nil {
		l.Error("Failed to grab last check for user's preferred game mode")
		captureError(err)
		return
	}

	l.Log("Determining if the last check was done within the last 6 hours")
	if time.Now().Unix()-lastCheck.DateChecked > 10800 {
		l.Log("Last check was made more than 3 hours ago, fetching new data")
		data, err := postingAPI.GetUser(osuapi.M{"u": dbOsuPlayer.UserID, "m": strconv.Itoa(prosuUser.OsuSettings.Mode)})
		if err != nil {
			l.Error("Failed to grab new data")
			captureError(err)
			return
		}
		request := createRequest(dbOsuPlayer.GetId(), data)

		// Save the request
		l.Log("We got the data, now we have to save the request to the database")
		err = connection.Collection("osurequestmodels").Save(request)
		if err != nil {
			l.Error("Failed to save the request")
			captureError(err)
			return
		}

		// Now we add the ID of the request to the checks field
		checks = append(checks, request.GetId())
		if prosuUser.OsuSettings.Mode == 0 {
			dbOsuPlayer.Modes.Standard.Checks = checks
		} else if prosuUser.OsuSettings.Mode == 1 {
			dbOsuPlayer.Modes.Taiko.Checks = checks
		} else if prosuUser.OsuSettings.Mode == 2 {
			dbOsuPlayer.Modes.CTB.Checks = checks
		} else if prosuUser.OsuSettings.Mode == 3 {
			dbOsuPlayer.Modes.Mania.Checks = checks
		}

		// Now we save the updated osuPlayer
		l.Log("Appended new data to the player's document. Saving")
		err = connection.Collection("osuplayermodels").Save(dbOsuPlayer)
		if err != nil {
			l.Error("Failed to the updated player document")
			captureError(err)
			return
		}
		l.Log("Saved the updated player document. Now we can generate the image.")
		postImage, err := generateImage(prosuUser, dbOsuPlayer, checks, l)
		var postImageBuffer bytes.Buffer
		png.Encode(&postImageBuffer, postImage)
		postImageBase64 := base64.StdEncoding.EncodeToString(postImageBuffer.Bytes())

		if err != nil {
			l.Error("Failed to generate image for user")
			captureError(err)
			return
		}
		l.Log("Image has now been generated. Testing if user's Twitter credentials are valid")
		prosuTwitter := anaconda.NewTwitterApiWithCredentials(prosuUser.Twitter.Token, prosuUser.Twitter.TokenSecret, consumerKey, consumerSecret)
		ok, err := prosuTwitter.VerifyCredentials()
		if err != nil {
			l.Error("Failed to check validity of Twitter credentials")
			captureError(err)
			return
		}
		if !ok {
			l.Error("Twitter credentials were not valid. Disabling tweets for user")
			prosuUser.OsuSettings.Enabled = false
			err = connection.Collection("usermodels").Save(prosuUser)
			if err != nil {
				l.Error("Failed to disable user's tweets")
				captureError(err)
				return
			}
			l.Log("Successfully disabled user's tweets after realizing their credentials are invalid")
			return
		}
		l.Log("User's credentials are valid. Uploading media")

		media, err := prosuTwitter.UploadMedia(postImageBase64)
		if err != nil {
			l.Error("Failed to upload image to Twitter")
			captureError(err)
			return
		}
		l.Log("Successfully uploaded image to Twitter. Creating Tweet")
		urlVals := url.Values{}
		urlVals.Add("media_ids", media.MediaIDString)
		tweet, err := prosuTwitter.PostTweet("osu! stats automatically generated by https://prosu.xyz", urlVals)
		if err != nil {
			l.Error("Error posting tweet")
			captureError(err)
			return
		}
		l.Log("Tweet successfully posted: https://twitter.com/" + prosuUser.Twitter.Profile.Handle + "/" + tweet.IdStr)
		l.Log("Adding to database")

		dbTweet := UserTweet{
			DatePosted: time.Now().Unix(),
			TweetObject: TweetObject{
				ID: tweet.IdStr,
			},
		}

		prosuUser.TweetHistory = append(prosuUser.TweetHistory, dbTweet)
		err = connection.Collection("usermodels").Save(prosuUser)
		if err != nil {
			l.Error("Failed to add new tweet to database")
			captureError(err)
			return
		}
		l.Log("Successfully added new tweet to user's profile. Tweet posting complete!")
	} else {
		l.Log("Last check was made less than 3 hours ago, we don't need new data")
	}
}

// For logging during posting
type pLogger struct {
	UserID string
}

func (pL pLogger) Log(msg string) {
	log.Debug("[POSTING: " + pL.UserID + "] " + msg)
}

func (pL pLogger) Error(msg string) {
	log.Error("[POSTING: " + pL.UserID + "] " + msg)
}

// Create an OsuRequest from api data
func createRequest(dbID bson.ObjectId, data *osuapi.User) *OsuRequest {
	return &OsuRequest{
		OsuPlayer:   dbID,
		DateChecked: time.Now().Unix(),
		Data: OsuRequestData{
			PlayerID:   data.UserID,
			PlayerName: data.Username,
			Counts: requestDataCounts{
				Count50s:  data.Count50,
				Count100s: data.Count100,
				Count300s: data.Count300,
				SS:        data.CountRankSS,
				SSH:       data.CountRankSSH,
				S:         data.CountRankS,
				SH:        data.CountRankSH,
				A:         data.CountRankA,
				Plays:     data.Playcount,
			},
			Scores: requestDataScores{
				Ranked: data.RankedScore,
				Total:  data.TotalScore,
			},
			PP: requestDataPP{
				Raw:         data.PP,
				Rank:        data.GlobalRank,
				CountryRank: data.CountryRank,
			},
			Country:  data.Country,
			Level:    data.Level,
			Accuracy: data.Accuracy,
		},
	}
}

// Grab the user's avatar
func getAvatar(userID string) (image.Image, error) {
	res, err := http.Get("https://a.ppy.sh/" + userID)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func generateImage(user *User, player *OsuPlayer, checks []bson.ObjectId, l pLogger) (image.Image, error) {
	previousRequest := &OsuRequest{}
	newRequest := &OsuRequest{}
	l.Log("Grabbing previous requests")
	err := connection.Collection("osurequestmodels").FindById(checks[len(checks)-2], previousRequest)
	if err != nil {
		l.Error("Failed to grab old request")
		return nil, err
	}
	err = connection.Collection("osurequestmodels").FindById(checks[len(checks)-1], newRequest)
	if err != nil {
		l.Error("Failed to grab new request")
		return nil, err
	}

	l.Log("Successfully grabbed previous requests. Grabbing avatar")
	avatar, err := getAvatar(player.UserID)
	if err != nil {
		l.Error("Failed to grab avatar")
		return nil, err
	}

	// Start to generate the image

	// Load mode image
	var modeImage image.Image
	var imageFileName string
	if user.OsuSettings.Mode == 0 {
		imageFileName = "./assets/modes/osu.png"
	} else if user.OsuSettings.Mode == 1 {
		imageFileName = "./assets/modes/taiko.png"
	} else if user.OsuSettings.Mode == 2 {
		imageFileName = "./assets/modes/ctb.png"
	} else if user.OsuSettings.Mode == 3 {
		imageFileName = "./assets/modes/mania.png"
	}

	file, err := ioutil.ReadFile(imageFileName)
	if err != nil {
		l.Error("Failed to load mode image")
		return nil, err
	}
	modeImage, _, err = image.Decode(bytes.NewReader(file))
	if err != nil {
		l.Error("Failed to decode mode image")
		return nil, err
	}

	// Load flag image
	var flagImage image.Image
	file, err = ioutil.ReadFile("./assets/flags/" + strings.ToUpper(newRequest.Data.Country) + ".png")
	if err != nil {
		if os.IsNotExist(err) {
			l.Error("Flag for country " + newRequest.Data.Country + " doesn't exist. Inserting blank flag")
			blankFile, newErr := ioutil.ReadFile("./assets/flags/__.png")
			if newErr != nil {
				l.Error("Failed to load blank flag")
				return nil, newErr
			}
			file = blankFile
		}
	}
	flagImage, _, decodeErr := image.Decode(bytes.NewReader(file))
	if decodeErr != nil {
		l.Error("Failed to decode flag")
		return nil, decodeErr
	}

	// Create the context
	dc := gg.NewContext(440, 220)

	// Create background
	dc.SetRGB(0, 0, 0)
	dc.Clear()

	// Draw the avatar
	dc.DrawImage(resize.Resize(100, 100, avatar, resize.Lanczos3), 0, 0)

	// Draw mode
	dc.DrawImage(resize.Resize(45, 45, modeImage, resize.Lanczos3), 25, 160)

	// Draw country flag
	dc.DrawImage(resize.Resize(45, 30, flagImage, resize.Lanczos3), 25, 115)

	// We don't need these anymore
	avatar = nil
	modeImage = nil
	flagImage = nil

	/* Draw player info */

	// Stats For:
	dc.SetFontFace(arialFont20)
	dc.SetFillStyle(gg.NewSolidPattern(color.White))
	dc.DrawString("Stats For: ", 110, 24)
	statsStringSizeW, _ := dc.MeasureString("Stats For: ")

	// Player Name
	dc.SetFontFace(arialBold20)
	dc.DrawString(player.PlayerName, 110+statsStringSizeW, 24)

	// Updated On:
	dc.SetFontFace(arialFont12)
	updatedTime := time.Unix(newRequest.DateChecked, 0)
	dc.DrawString("Updated On: "+updatedTime.Month().String()+" "+strconv.Itoa(updatedTime.Day())+", "+strconv.Itoa(updatedTime.Year()), 110, 40)

	// Create line under date
	dc.DrawLine(100, 45, 440, 45)
	dc.Stroke()

	/* Start drawing the actual data */
	vert := 63.00

	// Rank
	newRankData := newRequest.Data.PP.Rank
	oldRankData := previousRequest.Data.PP.Rank
	difference := float64(newRankData - oldRankData)
	arrow := 0
	if difference < 0 {
		difference *= -1
		arrow = 1
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 0
	}
	str := "Rank: " + formatDecimal(float64(newRankData))
	dc.DrawString(str, 110.00, vert)
	width, _ := dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// Country Rank
	newRankData = newRequest.Data.PP.CountryRank
	oldRankData = previousRequest.Data.PP.CountryRank
	difference = float64(newRankData - oldRankData)
	if difference < 0 {
		difference *= -1
		arrow = 1
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 0
	}
	str = "Country Rank: " + formatDecimal(float64(newRankData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// PP
	newPPData := newRequest.Data.PP.Raw
	oldPPData := previousRequest.Data.PP.Raw
	difference = float64(newPPData - oldPPData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "PP: " + formatDecimal(float64(newPPData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// Play Count
	newPlayCountData := newRequest.Data.Counts.Plays
	oldPlayCountData := previousRequest.Data.Counts.Plays
	difference = float64(newPlayCountData - oldPlayCountData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "Play Count: " + formatDecimal(float64(newPlayCountData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// Level
	newLevelData := newRequest.Data.Level
	oldLevelData := previousRequest.Data.Level
	difference = float64(newLevelData - oldLevelData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "Level: " + formatDecimal(float64(newLevelData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	//Accuracy
	newAccData := newRequest.Data.Accuracy
	oldAccData := previousRequest.Data.Accuracy
	difference = float64(newAccData - oldAccData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "Accuracy: " + formatDecimal(float64(newAccData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// SS
	newSSData := newRequest.Data.Counts.SS + newRequest.Data.Counts.SSH
	oldSSData := previousRequest.Data.Counts.SS + previousRequest.Data.Counts.SSH
	difference = float64(newSSData - oldSSData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "SS: " + formatDecimal(float64(newSSData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// S
	newSData := newRequest.Data.Counts.S + newRequest.Data.Counts.SH
	oldSData := previousRequest.Data.Counts.S + previousRequest.Data.Counts.SH
	difference = float64(newSData - oldSData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "S: " + formatDecimal(float64(newSData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	// A
	newAData := newRequest.Data.Counts.A
	oldAData := previousRequest.Data.Counts.A
	difference = float64(newAData - oldAData)
	if difference < 0 {
		difference *= -1
		arrow = 0
	} else if difference > 0 {
		arrow = -1
	} else {
		arrow = 1
	}
	str = "A: " + formatDecimal(float64(newAData))
	dc.DrawString(str, 110.00, vert)
	width, _ = dc.MeasureString(str)
	drawDifference(dc, difference, vert, arrow, width)
	vert += 18

	return dc.Image(), nil
}

var colorRed = gg.NewSolidPattern(color.RGBA{
	R: 255,
	G: 0,
	B: 0,
	A: 255,
})

var colorGreen = gg.NewSolidPattern(color.RGBA{
	R: 0,
	G: 255,
	B: 0,
	A: 255,
})

var colorGray = gg.NewSolidPattern(color.RGBA{
	R: 128,
	G: 128,
	B: 128,
	A: 255,
})

// Round to 0.10
func formatDecimal(x float64) string {
	rounded := math.Floor(x)
	decimal := math.Floor((x - rounded) * 100)
	var decimalString string
	if decimal == 0 {
		decimalString = ""
	} else {
		decimalString = "." + strconv.Itoa(int(decimal))
	}

	return humanize.Comma(int64(rounded)) + decimalString
}

// Draws the specified color arrow
func drawDifference(dc *gg.Context, difference, height float64, arrow int, textWidth float64) {
	diffString := formatDecimal(difference)

	if arrow == -1 {
		dc.SetFillStyle(colorRed)

		dc.MoveTo(textWidth+110+5, height-8.5)
		dc.LineTo(textWidth+110+20, height-8.5)
		dc.LineTo(textWidth+110+12.5, height)
		dc.Fill()

		dc.DrawString(diffString, 110+textWidth+22, height)
	} else if arrow == 0 {
		dc.SetFillStyle(colorGray)

		dc.MoveTo(110+textWidth+5, height-5.5)
		dc.LineTo(110+textWidth+20, height-5.5)
		dc.LineTo(110+textWidth+12.5, height-13)
		dc.Fill()

		dc.MoveTo(110+textWidth+5, height-5.5)
		dc.LineTo(110+textWidth+20, height-5.5)
		dc.LineTo(110+textWidth+12.5, height+2)
		dc.Fill()

		dc.DrawString(diffString, 110+textWidth+20, height)
	} else {
		dc.SetFillStyle(colorGreen)

		dc.MoveTo(110+textWidth+5, height-2.5)
		dc.LineTo(110+textWidth+20, height-2.5)
		dc.LineTo(110+textWidth+12.5, height-11)

		dc.DrawString(diffString, 110+textWidth+20, height)
	}
}
