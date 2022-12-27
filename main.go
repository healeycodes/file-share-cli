package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const MAX_UPLOAD_SIZE = 1024 * 1024 * 1024 // 1GB
const DEFAULT_PORT = "4000"

type application struct {
	website string
	auth    struct {
		username string
		password string
	}
}

func main() {
	app := new(application)

	port := os.Getenv("PORT")
	if port == "" {
		port = DEFAULT_PORT
	}
	website := os.Getenv("RAILWAY_STATIC_URL")
	if website == "" {
		app.website = "http://localhost:" + port
	} else {
		app.website = "https://" + website
	}

	app.auth.username = os.Getenv("AUTH_USERNAME")
	app.auth.password = os.Getenv("AUTH_PASSWORD")
	if app.auth.username == "" {
		log.Fatal("basic auth username must be provided")
	}
	if app.auth.password == "" {
		log.Fatal("basic auth password must be provided")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/dl", app.download)
	mux.HandleFunc("/upload", app.basicAuth(app.upload))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("starting server on %s", srv.Addr)
	err := srv.ListenAndServe()
	log.Fatal(err)
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `# if you are me, copy this to your ~/.bashrc
# and use it like this: share somefile.txt
# a download link will be echoed
# don't forget to replace user/pass!
function share () {
	curl -u user:pass -F "file=@$1" %v/upload
}`, app.website)
}

func (app *application) download(w http.ResponseWriter, r *http.Request) {
	fp := r.URL.Query().Get("f")
	if fp == "" {
		http.Error(w, "Missing query parameter e.g. `?f=examplefile.txt`", http.StatusBadRequest)
		return
	}

	// Make the browser open a download dialog
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(fp))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Avoid directory traversal (https://dzx.cz/2021/04/02/go_path_traversal/)
	http.ServeFile(w, r, filepath.Join("uploads", filepath.Join("/", fp)))
}

// https://freshman.tech/file-upload-golang/
func (app *application) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(w, fmt.Sprintf("File too large. Must be smaller than %v bytes", MAX_UPLOAD_SIZE), http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create the uploads folder if it doesn't already exist
	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create file
	dst, err := os.Create(fmt.Sprintf("./uploads/%v", fileHeader.Filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Write to file
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reply with a download link
	w.Write([]byte(fmt.Sprintf("%v/dl?f=%v", app.website, fileHeader.Filename)))
}

// https://www.alexedwards.net/blog/basic-authentication-in-go
func (app *application) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(app.auth.username))
			expectedPasswordHash := sha256.Sum256([]byte(app.auth.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
