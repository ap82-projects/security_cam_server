package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	socketio "github.com/googollee/go-socket.io"
	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/cors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// func handleRequests() {
// 	myRouter := mux.NewRouter().StrictSlash(true)
// 	myRouter.HandleFunc((/))
// }

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// socket.io

	mux := http.NewServeMux()
	mux.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"hello\": \"world\"}"))
	})

	server := socketio.NewServer(nil)

	// server.OnConnect("/", func(s socketio.Conn) error {
	server.OnConnect("/socketio", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("Socket connected:", s.ID())
		return nil
	})

	// server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
	server.OnEvent("/socketio", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	// server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
	server.OnEvent("/socketio", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	// server.OnEvent("/", "bye", func(s socketio.Conn) string {
	server.OnEvent("/socketio", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)
	})

	go server.Serve()
	defer server.Close()

	// http.Handle("/socket.io/", server)
	mux.Handle("/", server)

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

	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)

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
	type User struct {
		Name      string
		GoogleId  string
		Email     string
		Phone     string
		Incidents []Incident
		Watching  bool
	}

	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
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
			userGoogleId := r.FormValue("id")
			log.Println("Retrieving user with Google ID", userGoogleId)
			query := client.Collection("users").Where("googleid", "==", userGoogleId).Documents(ctx)
			for {
				doc, err := query.Next()
				if err == iterator.Done {
					break
				}

				user, err := json.Marshal(doc.Data())
				if err != nil {
					log.Println("Error:", err)
				}
				log.Println("User found. Sending response.")
				log.Println(string(user))
				fmt.Fprintln(w, string(user))
			}

		}

		if r.Method == "DELETE" {
			log.Println("Request type: DELETE")
			userId := r.FormValue("id")

			log.Println("Deleting user ID", userId)
			fsDeleteTime, err := client.Collection("users").Doc(userId).Delete(ctx)
			if err != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Println("An error has occurred:", err)
			} else {
				log.Println("User", userId, "deleted at", fsDeleteTime)
			}
		}
	})

	// http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
	// 	log.Println("Responding to call on /hello")
	// 	if r.URL.Path != "/hello" {
	// 		http.Error(w, "404 not found. Try again :(", http.StatusNotFound)
	// 		return
	// 	}

	// 	if r.Method != "GET" {
	// 		http.Error(w, "Method is not supported. Don't be so greedy.", http.StatusNotFound)
	// 		return
	// 	}

	// 	fmt.Fprintf(w, "Hello There!")
	// })

	mux.HandleFunc("/api/user/incident", func(w http.ResponseWriter, r *http.Request) {
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
				// Handle any errors in an appropriate way, such as returning them.
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
				// Handle any errors in an appropriate way, such as returning them.
				log.Println("An error has occurred:", err)
			}
		}
	})

	mux.HandleFunc("/api/user/watching", func(w http.ResponseWriter, r *http.Request) {
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
				// Handle any errors in an appropriate way, such as returning them.
				log.Println("An error has occurred:", err)
			}
		}
	})

	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Responding to call on /form")
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}

		fmt.Fprintf(w, "POST request successful\n")

		////////////////////////////////////////////////////////////
		// The Following Block pulls form the body of the request //
		type NameAdd struct {
			Name    string
			Address string
		}
		var na NameAdd
		err := json.NewDecoder(r.Body).Decode(&na)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "name: ", na.Name, "\n")
		fmt.Fprint(w, "address: ", na.Address, "\n")
		////////////////////////////////////////////////////////////

		////////////////////////////////////////////////////////////
		// The following block pulls from params of the request   //
		// (http://...?name=myname&address=nyaddress)             //
		name := r.FormValue("name")
		address := r.FormValue("address")

		fmt.Fprintf(w, "Name = %s\n", name)
		fmt.Fprintf(w, "Address = %s\n", address)
		////////////////////////////////////////////////////////////
	})

	handler := cors.Default().Handler(mux)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	handler = c.Handler(handler)

	log.Println("Starting server at port 8080")
	// handler := cors.Default().Handler(http)
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
