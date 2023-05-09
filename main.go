package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"

	"context"

	vision "cloud.google.com/go/vision/apiv1"
)

// array of swear words
var swearWords = []string{
	"fuck",
	"shit",
	"damn",
	"ass",
	"penis",
	"crypto",
	"cunt",
}

func main() {
	app := pocketbase.New()

	// oauth (before)
	app.OnRecordBeforeAuthWithOAuth2Request().Add(func(e *core.RecordAuthWithOAuth2Event) error {
		log.Println("OAuth before")
		log.Println(e.Record) // could be nil
		log.Println(e.OAuth2User)

		return nil
	})

	// oauth (after)
	app.OnRecordAfterAuthWithOAuth2Request().Add(func(e *core.RecordAuthWithOAuth2Event) error {
		log.Println("OAuth after")
		log.Println(e.Record)
		log.Println(e.OAuth2User)

		if e.IsNewRecord {
			id := e.Record.GetId()
			log.Println("user record created", id)

			// create userHealth record for new user
			healthCollection, err := app.Dao().FindCollectionByNameOrId("userHealth")
			if err != nil {
				log.Println(err)
			}

			// get image from https://api.dicebear.com/6.x/miniavs/svg?seed={id}
			url := "https://api.dicebear.com/6.x/miniavs/png?seed=" + id
			log.Println(url)

			// use http.get
			resp, err := http.Get(url)
			if err != nil {
				log.Println("error getting pfp:", err)
			}
			defer resp.Body.Close()

			// get filesystem
			fs, err := app.NewFilesystem()
			if err != nil {
				log.Println("error getting file system", err)
				return err
			}
			defer fs.Close()

			// get resp.body bytes
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("error reading body", err)
			}

			// upload file
			fileKey := e.Record.BaseFilesPath() + "/avatar.png"
			err = fs.Upload(body, fileKey)
			if err != nil {
				log.Println("error uploading file", err)
				return err
			}
			e.Record.Set("avatar", "avatar.png")

			// save record
			if err := app.Dao().SaveRecord(e.Record); err != nil {
				log.Println(err)
			}

			healthRecord := models.NewRecord(healthCollection)
			healthRecord.Set("user", id)
			healthRecord.Set("height", 0)
			healthRecord.Set("weight", 0)
			healthRecord.Set("goal", 0)
			healthRecord.Set("dailyAllowance", 0)
			healthRecord.Set("isFemale", false)
			healthRecord.Set("age", 0)
			healthRecord.Set("points", 0)
			healthRecord.Set("food", "{\"pantry\":[],\"plans\":[]}")

			if err := app.Dao().SaveRecord(healthRecord); err != nil {
				log.Println(err)
			}
		}

		return nil
	})

	// record create (before)
	app.OnRecordBeforeCreateRequest().Add(func(e *core.RecordCreateEvent) error {
		log.Println(e.Record) // still unsaved

		if e.Collection.Name == "progressPosts" {
			title := e.Record.GetString("title")
			description := e.Record.GetString("description")

			filteredTitle := title
			for _, word := range swearWords {
				filteredTitle = strings.ReplaceAll(filteredTitle, word, "****")
			}
			log.Print("Filtered title: " + filteredTitle)
			e.Record.Set("title", filteredTitle)

			filteredDesc := description
			for _, word := range swearWords {
				filteredDesc = strings.ReplaceAll(filteredDesc, word, "****")
			}
			log.Print("Filtered description: " + filteredDesc)
			e.Record.Set("description", filteredDesc)

		}

		return nil
	})

	// record create (after)
	app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
		log.Println(e.Record) // still unsaved

		if e.Collection.Name == "comments" {
			// create a new record in the notifications collection
			collection, err := app.Dao().FindCollectionByNameOrId("notifications")
			if err != nil {
				log.Println(err)
			}

			// get relationship record (post)
			post, err := app.Dao().FindRecordById("progressPosts", e.Record.GetString("post"))
			if err != nil {
				log.Println(err)
			}

			record := models.NewRecord(collection)
			record.Set("viewed", false)
			record.Set("for", post.GetString("creator"))
			record.Set("message", false)
			record.Set("post", e.Record.Get("post"))

			if err := app.Dao().SaveRecord(record); err != nil {
				log.Println(err)
			}
		} else if e.Collection.Name == "progressPosts" {
			if e.Record.GetString("picture") != "" {
				fileKey := e.Record.BaseFilesPath() + "/" + e.Record.GetString("picture")

				fs, err := app.NewFilesystem()
				if err != nil {
					return err
				}
				defer fs.Close()

				// retrieve file
				f, err := fs.GetFile(fileKey)
				if err != nil {
					return err
				}
				defer f.Close()

				// use Google Vision API to detect labels
				ctx := context.Background()
				client, err := vision.NewImageAnnotatorClient(ctx)
				if err != nil {
					return err
				}
				defer client.Close()

				image, err := vision.NewImageFromReader(f)
				if err != nil {
					return err
				}
				props, err := client.DetectSafeSearch(ctx, image, nil)
				if err != nil {
					return err
				}
				log.Println("IMAGE VISION:", props.String())

				// if image is not safe, delete it
				if props.Adult > 2 || props.Medical > 3 || props.Violence > 2 || props.Racy > 3 {
					log.Println("Image is nsfw, deleting...")
					f.Close()
					client.Close()
					e.Record.Set("picture", "")

					// save
					if err := app.Dao().SaveRecord(e.Record); err != nil {
						log.Println(err)
					}

					// delete file
					if err := fs.Delete(fileKey); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
