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

		if e.Collection.Name == "users" {
			log.Println("user record created")
			id := e.Record.GetId()

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
				log.Println(err)
			}
			defer resp.Body.Close()

			// get filesystem
			fs, err := app.NewFilesystem()
			if err != nil {
				return err
			}
			defer fs.Close()

			// get resp.body bytes
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}

			// upload file
			fileKey := e.Record.BaseFilesPath() + "/avatar.png"
			err = fs.Upload(body, fileKey)
			if err != nil {
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
		} else if e.Collection.Name == "comments" {
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

/* database structure
[
    {
        "id": "wuhipjbdln44ccv",
        "name": "friendRequests",
        "type": "base",
        "system": false,
        "schema": [
            {
                "id": "jazrehmb",
                "name": "from",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            },
            {
                "id": "qrmt3a03",
                "name": "to",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            }
        ],
        "indexes": [
            "CREATE INDEX `idx_AkKnmYP` ON `friendRequests` (`to`)"
        ],
        "listRule": "to.id = @request.auth.id",
        "viewRule": "to.id = @request.auth.id",
        "createRule": "(@request.data.from.id = @request.auth.id) && (to.blocked !~ @request.data.from)",
        "updateRule": null,
        "deleteRule": "to.id = @request.auth.id",
        "options": {}
    },
    {
        "id": "pr1kmf9gphhrs6v",
        "name": "comments",
        "type": "base",
        "system": false,
        "schema": [
            {
                "id": "qffurj5z",
                "name": "comment",
                "type": "text",
                "system": false,
                "required": true,
                "options": {
                    "min": null,
                    "max": null,
                    "pattern": ""
                }
            },
            {
                "id": "efkdsvq1",
                "name": "createdBy",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": true,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            },
            {
                "id": "ubkkrmgl",
                "name": "isBot",
                "type": "bool",
                "system": false,
                "required": false,
                "options": {}
            },
            {
                "id": "zjgl7h5x",
                "name": "post",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "ggat2jqfoxls4l0",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            }
        ],
        "indexes": [
            "CREATE INDEX `idx_RvQrRnC` ON `comments` (`created`)",
            "CREATE INDEX `idx_VxGqrDL` ON `comments` (`post`)"
        ],
        "listRule": "(@request.auth.id = createdBy.id) && (createdBy.blocked !~ @request.auth.id) && (@request.auth.blocked !~ createdBy)",
        "viewRule": "(@request.auth.id = createdBy.id) && (createdBy.blocked !~ @request.auth.id) && (@request.auth.blocked !~ createdBy)",
        "createRule": "(@request.auth.id = createdBy.id) && (createdBy.blocked !~ @request.auth.id)",
        "updateRule": "@request.auth.id = createdBy.id",
        "deleteRule": "@request.auth.id = createdBy.id || @request.auth.id = post.creator.id",
        "options": {}
    },
    {
        "id": "ggat2jqfoxls4l0",
        "name": "progressPosts",
        "type": "base",
        "system": false,
        "schema": [
            {
                "id": "y6frqzrv",
                "name": "title",
                "type": "text",
                "system": false,
                "required": true,
                "options": {
                    "min": null,
                    "max": null,
                    "pattern": ""
                }
            },
            {
                "id": "mowdwbml",
                "name": "description",
                "type": "text",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null,
                    "pattern": ""
                }
            },
            {
                "id": "msrca8bx",
                "name": "picture",
                "type": "file",
                "system": false,
                "required": false,
                "options": {
                    "maxSelect": 1,
                    "maxSize": 5242880,
                    "mimeTypes": [
                        "image/jpeg",
                        "image/png",
                        "image/svg+xml",
                        "image/gif",
                        "image/webp"
                    ],
                    "thumbs": [],
                    "protected": false
                }
            },
            {
                "id": "l6hyzpl7",
                "name": "creator",
                "type": "relation",
                "system": false,
                "required": true,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            },
            {
                "id": "a5e3km4p",
                "name": "friendsOnly",
                "type": "bool",
                "system": false,
                "required": false,
                "options": {}
            }
        ],
        "indexes": [
            "CREATE INDEX `idx_8jpKgyx` ON `progressPosts` (`created`)",
            "CREATE INDEX `idx_8UiwydN` ON `progressPosts` (`creator`)"
        ],
        "listRule": "(@request.auth.id != '') && (@request.auth.blocked !~ creator) && (creator.blocked !~ @request.auth.id)",
        "viewRule": "(@request.auth.id != '') && (@request.auth.blocked !~ creator) && (creator.blocked !~ @request.auth.id)",
        "createRule": "@request.auth.id = creator.id",
        "updateRule": "@request.auth.id = creator.id",
        "deleteRule": "@request.auth.id = creator.id",
        "options": {}
    },
    {
        "id": "lp99wjagmq2x8on",
        "name": "notifications",
        "type": "base",
        "system": false,
        "schema": [
            {
                "id": "pzavdm8k",
                "name": "post",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "ggat2jqfoxls4l0",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            },
            {
                "id": "fi02d4hv",
                "name": "for",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            },
            {
                "id": "dxum9xhe",
                "name": "message",
                "type": "text",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null,
                    "pattern": ""
                }
            },
            {
                "id": "j7q70egc",
                "name": "viewed",
                "type": "bool",
                "system": false,
                "required": false,
                "options": {}
            }
        ],
        "indexes": [
            "CREATE INDEX `idx_EY8J8Dk` ON `notifications` (`for`)"
        ],
        "listRule": "for.id = @request.auth.id",
        "viewRule": "for.id = @request.auth.id",
        "createRule": null,
        "updateRule": "for.id = @request.auth.id",
        "deleteRule": "for.id = @request.auth.id",
        "options": {}
    },
    {
        "id": "_pb_users_auth_",
        "name": "users",
        "type": "auth",
        "system": false,
        "schema": [
            {
                "id": "users_name",
                "name": "name",
                "type": "text",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null,
                    "pattern": ""
                }
            },
            {
                "id": "users_avatar",
                "name": "avatar",
                "type": "file",
                "system": false,
                "required": false,
                "options": {
                    "maxSelect": 1,
                    "maxSize": 5242880,
                    "mimeTypes": [
                        "image/jpeg",
                        "image/png",
                        "image/svg+xml",
                        "image/gif",
                        "image/webp"
                    ],
                    "thumbs": null,
                    "protected": false
                }
            },
            {
                "id": "vhgembnx",
                "name": "friends",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": null,
                    "displayFields": []
                }
            },
            {
                "id": "tljanysn",
                "name": "blocked",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": null,
                    "displayFields": []
                }
            },
            {
                "id": "6pfilyih",
                "name": "bookmarks",
                "type": "relation",
                "system": false,
                "required": false,
                "options": {
                    "collectionId": "ggat2jqfoxls4l0",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": null,
                    "displayFields": []
                }
            }
        ],
        "indexes": [],
        "listRule": "blocked !~ @request.auth.id",
        "viewRule": "(id = @request.auth.id || friends.id ~ @request.auth.id) && (blocked !~ @request.auth.id)",
        "createRule": "",
        "updateRule": "id = @request.auth.id",
        "deleteRule": "id = @request.auth.id",
        "options": {
            "allowEmailAuth": false,
            "allowOAuth2Auth": true,
            "allowUsernameAuth": false,
            "exceptEmailDomains": null,
            "manageRule": null,
            "minPasswordLength": 8,
            "onlyEmailDomains": null,
            "requireEmail": false
        }
    },
    {
        "id": "flox0132evc94iv",
        "name": "userHealth",
        "type": "base",
        "system": false,
        "schema": [
            {
                "id": "dpibtdlg",
                "name": "user",
                "type": "relation",
                "system": false,
                "required": true,
                "options": {
                    "collectionId": "_pb_users_auth_",
                    "cascadeDelete": false,
                    "minSelect": null,
                    "maxSelect": 1,
                    "displayFields": []
                }
            },
            {
                "id": "5jsvbcwt",
                "name": "weight",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            },
            {
                "id": "5pzwzcrj",
                "name": "height",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            },
            {
                "id": "cxdqttvu",
                "name": "food",
                "type": "json",
                "system": false,
                "required": false,
                "options": {}
            },
            {
                "id": "r5ne7slv",
                "name": "points",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            },
            {
                "id": "8h8gmry1",
                "name": "dailyAllowance",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            },
            {
                "id": "euefvref",
                "name": "goal",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            },
            {
                "id": "jb8fcfve",
                "name": "isFemale",
                "type": "bool",
                "system": false,
                "required": false,
                "options": {}
            },
            {
                "id": "8e0zttb7",
                "name": "activity",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            },
            {
                "id": "m14r2er4",
                "name": "age",
                "type": "number",
                "system": false,
                "required": false,
                "options": {
                    "min": null,
                    "max": null
                }
            }
        ],
        "indexes": [
            "CREATE INDEX `idx_WwTuLHs` ON `userHealth` (`user`)"
        ],
        "listRule": null,
        "viewRule": "@request.auth.id = user.id",
        "createRule": "@request.auth.id = user.id",
        "updateRule": "@request.auth.id = user.id",
        "deleteRule": null,
        "options": {}
    }
]
*/
