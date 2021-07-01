package main

import (
	// "context"
	// "encoding/json"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	// "os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	// "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	gosocketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/cors"
)

// func setupCorsResponse(w *http.ResponseWriter, req *http.Request) {
// 	(*w).Header().Set("Access-Control-Allow-Origin", "*")
// 	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
// 	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")
// }

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := mux.NewRouter()
	// router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	// 	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Access-Control-Request-Headers, Access-Control-Request-Method, Connection, Host, Origin, User-Agent, Referer, Cache-Control, X-header")
	// })

	/////////////////////////////////////////////////////////////////////////////
	//*************************************************************************//
	//********************** Firestorm Authentication *************************//
	//*************************************************************************//
	/////////////////////////////////////////////////////////////////////////////

	credentialsJson := []byte(os.Getenv("FIRESTORE_JSON"))
	ctx := context.Background()
	sa := option.WithCredentialsJSON(credentialsJson)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	// This is for retrieving all users in database
	// iter := client.Collection("users").Documents(ctx)
	// for {
	// 	doc, err := iter.Next()
	// 	if err == iterator.Done {
	// 		break
	// 	}
	// 	if err != nil {
	// 		log.Fatalf("Failed to iterate: %v", err)
	// 	}
	// 	fmt.Println(doc.Data())
	// }

	/////////////////////////////////////////////////////////////////////////////
	//*************************************************************************//
	//****************************** Socket.io ********************************//
	//*************************************************************************//
	/////////////////////////////////////////////////////////////////////////////

	type Message struct {
		Text string
	}

	Server := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())
	fmt.Println("Socket Initialize...")

	// socket connection
	Server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		fmt.Println("Connected", c.Id())
		c.Join("Room")
	})

	// socket disconnection
	Server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		fmt.Println("Disconnected", c.Id())

		// handles when someone closes the tab
		c.Leave("Room")
	})

	// type Message struct {
	// 	Id string `json:"id"`
	// 	Watching string `json:"watching"`
	// }

	// //handle custom event
	// server.On("send", func(c *gosocketio.Channel, msg Message) string {
	// 	//send event to all in room
	// 	c.BroadcastTo("Room", "message", msg)
	// 	return "OK"
	// })

	// watch socket
	Server.On("/watch", func(c *gosocketio.Channel, message Message) string {
		log.Println("in watch socket")
		// fmt.Println(message.Text)
		c.BroadcastTo("Room", "/message", message.Text)
		return "message sent successfully."
	})

	// watch event
	Server.On("watch", func(c *gosocketio.Channel, msg Message) string {
		//send event to all in room
		log.Println("in watch event")
		c.BroadcastTo("Room", "message", msg)
		return "OK"
	})

	router.Handle("/socket.io/", Server)

	/////////////////////////////////////////////////////////////////////////////
	//*************************************************************************//
	//******************** Structs for interacting with DB ********************//
	//*************************************************************************//
	/////////////////////////////////////////////////////////////////////////////

	type UserId struct {
		Id string
	}
	type Incident struct {
		Time  string
		Image string
	}
	type WatchingUpdate struct {
		Watching bool
	}
	type WatchingMessageToCamera struct {
		Id       string
		Watching bool
	}
	type User struct {
		Name      string
		GoogleId  string
		Email     string
		Phone     string
		Incidents []Incident
		Watching  bool
	}

	/////////////////////////////////////////////////////////////////////////////
	//*************************************************************************//
	//************************** REST API Endpoints ***************************//
	//*************************************************************************//
	/////////////////////////////////////////////////////////////////////////////

	router.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Responding to call on /api/user")
		if r.Method == "POST" {
			log.Println("Request type: POST")

			var newUser User
			err := json.NewDecoder(r.Body).Decode(&newUser)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Println("Adding user:")
			log.Println(newUser)

			fsDocRef, fsWriteResult, err := client.Collection("users").Add(ctx, map[string]interface{}{
				"name":      newUser.Name,
				"googleid":  newUser.GoogleId,
				"email":     newUser.Email,
				"phone":     newUser.Phone,
				"incidents": newUser.Incidents,
				"watching":  newUser.Watching,
			})
			log.Println("New user id", fsDocRef.ID, "created at", fsWriteResult)
			fmt.Fprintln(w, "{ \"id\":", fsDocRef.ID, "}")
			if err != nil {
				log.Fatalf("Failed adding user: %v", err)
			}
		}

		if r.Method == "GET" {
			log.Println("Request type: GET")
			userDocId := r.FormValue("id")
			log.Println("Retrieving user with Document ID", userDocId)
			query, errQ := client.Collection("users").Doc(userDocId).Get(ctx)
			if errQ != nil {

			}
			var currentUser User
			mapstructure.Decode(query.Data(), &currentUser)
			currentUserData, err := json.Marshal(currentUser)
			if err != nil {
				log.Println("Error:", err)
			}
			log.Println("Sending user data")
			log.Println(string(currentUserData))
			fmt.Fprintln(w, string(currentUserData))
		}

		if r.Method == "DELETE" {
			log.Println("Request type: DELETE")
			userId := r.FormValue("id")

			log.Println("Deleting user ID", userId)
			fsDeleteTime, err := client.Collection("users").Doc(userId).Delete(ctx)
			if err != nil {
				log.Println("An error has occurred:", err)
			} else {
				log.Println("User", userId, "deleted at", fsDeleteTime)
			}
		}
	})

	router.HandleFunc("/api/user/google", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Request type: GET")
		userGoogleId := r.FormValue("id")
		log.Println("Retrieving user with Google ID", userGoogleId)
		query := client.Collection("users").Where("googleid", "==", userGoogleId).Documents(ctx)
		for {
			doc, err := query.Next()
			if err == iterator.Done {
				break
			}

			id, err := json.Marshal(doc.Ref.ID)
			if err != nil {
				log.Println("Error:", err)
			}
			docId := string(id)
			log.Println("User found. Sending response.")
			log.Println("Document id", docId)
			fmt.Fprintln(w, "{ \"id\":", docId, "}")
		}
	})

	router.HandleFunc("/api/user/incident", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Responding to call on /api/user/incident")

		if r.Method == "PUT" {
			log.Println("Request type: PUT")

			var newIncident Incident
			err := json.NewDecoder(r.Body).Decode(&newIncident)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Println("Received New Incident:")
			log.Println(newIncident)

			log.Println("Retrieving User Data")
			userId := r.FormValue("id")
			query, errQ := client.Collection("users").Doc(userId).Get(ctx)
			if errQ != nil {

			}

			var currentUser User
			mapstructure.Decode(query.Data(), &currentUser)
			log.Println(currentUser)
			log.Println("Updating User Data")
			currentUser.Incidents = append(currentUser.Incidents, newIncident)

			log.Println(currentUser)

			_, err = client.Collection("users").Doc(userId).Update(ctx, []firestore.Update{
				{
					Path:  "incidents",
					Value: currentUser.Incidents,
				},
			})
			if err != nil {
				log.Println("An error has occurred:", err)
			}
		}

		if r.Method == "DELETE" {
			userId := r.FormValue("id")
			incidentTime := r.FormValue("time")

			log.Println("Retrieving data for user ID", userId)
			query, errQ := client.Collection("users").Doc(userId).Get(ctx)
			if errQ != nil {

			}

			var currentUser User
			mapstructure.Decode(query.Data(), &currentUser)
			log.Println(currentUser)
			log.Println("Updating User Data")

			i := 0
			for _, incident := range currentUser.Incidents {
				if incident.Time != incidentTime {
					currentUser.Incidents[i] = incident
					i++
				}
			}
			currentUser.Incidents = currentUser.Incidents[:i]

			_, err = client.Collection("users").Doc(userId).Update(ctx, []firestore.Update{
				{
					Path:  "incidents",
					Value: currentUser.Incidents,
				},
			})
			if err != nil {
				log.Println("An error has occurred:", err)
			}
		}
	})

	router.HandleFunc("/api/user/watching", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Responding to call on /api/user/watching")

		if r.Method == "PUT" {
			log.Println("Request type: PUT")

			userId := r.FormValue("id")
			var currently WatchingUpdate
			err := json.NewDecoder(r.Body).Decode(&currently)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Println("User", userId, "current watching status:", currently.Watching)
			log.Println("Updating user data")
			_, err = client.Collection("users").Doc(userId).Update(ctx, []firestore.Update{
				{
					Path:  "watching",
					Value: currently.Watching,
				},
			})
			if err != nil {
				log.Println("An error has occurred:", err)
			}
			log.Println("Notifying Remote Camera")
			var message WatchingMessageToCamera
			message.Id = userId
			message.Watching = currently.Watching
			Server.BroadcastTo("Room", "watch", message)
		}
	})

	// /////////////////////////////////////////////////////////////////////////////
	// //*************************************************************************//
	// //**************** Server Initialization and Extra Stuff ******************//
	// //*************************************************************************//
	// /////////////////////////////////////////////////////////////////////////////

	// For serving static files
	// fileServer := http.FileServer(http.Dir("./static"))
	// http.Handle("/", fileServer)

	// Extra CORS rules just in case
	handler := cors.Default().Handler(router)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"POST", "GET", "PUT", "DELETE"},
	})
	handler = c.Handler(handler)

	fmt.Println("Server Started at http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil { // for use with above CORS rules
		// if err := http.ListenAndServe(":8080", router); err != nil { // for when not using the above CORS rules
		log.Fatal(err)
	}
}
