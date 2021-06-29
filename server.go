package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go"
	"github.com/joho/godotenv"
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

	// fmt.Println("client = ", reflect.TypeOf(client))

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
	type Report struct {
		Id   string
		Data Incident
	}
	type User struct {
		Name      string
		GoogleId  string
		Email     string
		Phone     string
		Incidents []Incident
	}

	http.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Responding to call on /api/user")
		if r.Method == "POST" {
			fmt.Println("Request type: POST")

			var newUser User
			err := json.NewDecoder(r.Body).Decode(&newUser)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println("Adding user:")
			fmt.Println(newUser)

			fsDocRef, fsWriteResult, err := client.Collection("users").Add(ctx, map[string]interface{}{
				"name":      newUser.Name,
				"googleid":  newUser.GoogleId,
				"email":     newUser.Email,
				"phone":     newUser.Phone,
				"incidents": newUser.Incidents,
			})
			fmt.Println("New user id", fsDocRef.ID, "created at", fsWriteResult)
			fmt.Fprintln(w, "{ \"id\":", fsDocRef.ID, "}")
			if err != nil {
				log.Fatalf("Failed adding user: %v", err)
			}
		}

		if r.Method == "GET" {
			fmt.Println("Request type: GET")
			userGoogleId := r.FormValue("id")
			fmt.Println("Retrieving user with Google ID", userGoogleId)
			query := client.Collection("users").Where("googleid", "==", userGoogleId).Documents(ctx)
			for {
				doc, err := query.Next()
				if err == iterator.Done {
					break
				}
				// if err != nil {
				//         return err
				// }
				// fmt.Println(doc.Data())
				// fmt.Fprintln(w, doc.Data())
				user, err := json.Marshal(doc.Data())
				if err != nil {
					fmt.Println("Error:", err)
				}
				fmt.Println("User found. Sending response.")
				fmt.Println(string(user))
				fmt.Fprintln(w, string(user))
			}
			// fmt.Fprintln(w, "User ID = ", userId)
			// fmt.Fprintln(w, query)
		}

		if r.Method == "PUT" {
			fmt.Println("Request type: PUT")

			var newReport Report
			err := json.NewDecoder(r.Body).Decode(&newReport)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println("Received New Report:")
			fmt.Println(newReport)

		}

		if r.Method == "DELETE" {
			fmt.Println("Request type: DELETE")
			userId := r.FormValue("id")
			// var deleteUser UserId
			// err := json.NewDecoder(r.Body).Decode(&deleteUser)
			// if err != nil {
			// http.Error(w, err.Error(), http.StatusBadRequest)
			// return
			// }
			fmt.Println("Deleting user ID", userId)
			fsDeleteTime, err := client.Collection("users").Doc(userId).Delete(ctx)
			if err != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Printf("An error has occurred: %s", err)
			} else {
				fmt.Println("User", userId, "deleted at", fsDeleteTime)
				// fmt.Fprintln(w, "User", userId, "deleted at", fsDeleteTime)
			}
		}
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Responding to call on /hello\n")
		if r.URL.Path != "/hello" {
			http.Error(w, "404 not found. Try again :(", http.StatusNotFound)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method is not supported. Don't be so greedy.", http.StatusNotFound)
			return
		}

		fmt.Fprintf(w, "Hello There!")
	})

	http.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Responding to call on /form\n")
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

	// http.HandleFunc("/api/user/{id}", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Println("Responding to call on /api/user/:id")
	// 	if r.Method == "GET" {
	// 		fmt.Println("Request type: GET")
	// 		// id := r.

	// 	}
	// })

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// rules_version = '2';
// service cloud.firestore {
//   match /databases/{database}/documents {
//     match /{document=**} {
//       allow read, write: if false;
//     }
//   }
// }
