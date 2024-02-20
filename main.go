package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	adminUsername = "admin"
	adminPassword = "pass"
)

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/upload", basicAuth(uploadHandler, adminUsername, adminPassword))
	http.HandleFunc("/images", imagesHandler)
	http.HandleFunc("/images/", viewImageHandler)
	http.HandleFunc("/admin", basicAuth(adminHandler, adminUsername, adminPassword))
	http.HandleFunc("/admin/delete/", basicAuth(deleteImageHandler, adminUsername, adminPassword))

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	http.ListenAndServe(":8088", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "home.html", nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	file, handler, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error Retrieving the File", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Generate UUID for filename
	uuid, _ := generateUUID()
	filename := uuid + filepath.Ext(handler.Filename)
	out, err := os.Create(filepath.Join("uploads", filename))
	if err != nil {
		http.Error(w, "Error Saving the File", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Error Saving the File", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func imagesHandler(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("uploads/*")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var images []string
	for _, file := range files {
		filename := filepath.Base(file)
		images = append(images, filename)
	}

	renderTemplate(w, "images.html", images)
}

func viewImageHandler(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/images/")
	http.ServeFile(w, r, filepath.Join("uploads", filename))
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("uploads/*")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var images []string
	for _, file := range files {
		filename := filepath.Base(file)
		images = append(images, filename)
	}

	renderTemplate(w, "admin.html", images)
}

func deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/admin/delete/")
	err := os.Remove(filepath.Join("uploads", filename))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func basicAuth(next http.HandlerFunc, username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Please enter your username and password for admin access."`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	tmpl = fmt.Sprintf("templates/%s", tmpl)
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func generateUUID() (string, error) {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		return "", err
	}
	// Set version (4) and variant (2)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
