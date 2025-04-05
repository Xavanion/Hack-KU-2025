package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"github.com/gorilla/mux"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

var users []User

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Telling the program the path to all relevant files

	// extract only the relevant part for us
	fmt.Println(r.URL.Path)
	file_name := r.URL.Path[len("/"):]
	fmt.Println(file_name)
	filepath := "Frontend/" 
	serveFile(w, r, filepath, file_name)
}

func serveFile(w http.ResponseWriter, r *http.Request, filepath string, file string) {
	//filepath := path.Clean(FilePath) // clean the file path from things like ..
	extension := r.Header.Get("Accept")
	fmt.Print(extension)
	//extension := path.Ext(filepath) // see if there is a file extension
	switch extension {
		case "application/json":
			filepath += "json/" + file
			w.Header().Set("Content-Type", "application/javascript")
		case "text/css":
			filepath += "css/" + file
			w.Header().Set("Content-Type", "text/css")
		default:
			if(file == ""){
				file = "index"	
			}
			filepath += "html/" + file + ".html"
			w.Header().Set("Content-Type", "text/html")
		}
	// clean the file path from things like ".." to prevent escapes
	filepath = path.Clean(filepath) 
	fmt.Print(filepath)
	// Check if the file exists in our system
    if _, err := os.Stat(filepath); os.IsNotExist(err) {
        http.NotFound(w, r)
        return
    }
	fmt.Print(filepath)
	// actually serve the file
	http.ServeFile(w, r, filepath)
}


func main() {
	fmt.Println("test")
	router := mux.NewRouter()
	//router.HandleFunc("/users", getUsers).Methods("GET")
	//router.HandleFunc("/users", createUser).Methods("POST")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
		case "/", "/index", "/home.js":
			handleHTTP(w, r);
		case "/about", "/info":
			break
		default:
			http.NotFound(w, r)
    }
})

	err := http.ListenAndServe(":8080", router)
	if(err != nil){
		fmt.Println("Error starting server:",err)
	}
}
