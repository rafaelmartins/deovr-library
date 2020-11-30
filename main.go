package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rafaelmartins/deovr-library/internal/deovr"
	"github.com/rafaelmartins/deovr-library/internal/gallery"
)

var data = &deovr.DeoVR{}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if err := gallery.Index(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func SceneHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if s := data.GetSceneByName(vars["scene"]); s == nil {
		http.NotFound(w, r)
	} else if err := gallery.Scene(w, s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func DeoVRHandler(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func MediaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if f, err := data.GetMediaPath(vars["scene"], vars["file"]); err != nil {
		http.NotFound(w, r)
	} else {
		http.ServeFile(w, r, f)
	}
}

func ThumbHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if f, err := data.GetThumbnailPath(vars["scene"], vars["file"]); err != nil {
		http.NotFound(w, r)
	} else {
		http.ServeFile(w, r, f)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: deovr-library hostname:port label=directory [label=directory ...]\n")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}

	pieces := strings.Split(os.Args[1], ":")
	if pieces[0] == "" {
		fmt.Fprintf(os.Stderr, "Error: hostname required for URL generation\n\n")
		usage()
	}

	if _, err := strconv.Atoi(pieces[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: port must be an integer\n\n")
		usage()
	}

	for _, dir := range os.Args[2:] {
		p := strings.SplitN(dir, "=", 2)
		if len(p) != 2 {
			fmt.Fprintf(os.Stderr, "Error: missing name for directory: %s\n\n", dir)
			usage()
		}
		if err := data.LoadScene(p[0], p[1], os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to load scene: %s\n\n", err)
			usage()
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/deovr", DeoVRHandler)
	r.HandleFunc("/scene/{scene}", SceneHandler)
	r.HandleFunc("/media/{scene}/{file}", MediaHandler)
	r.HandleFunc("/thumb/{scene}/{file}", ThumbHandler)

	fmt.Fprintf(os.Stderr, "\n * Running on http://%s/\n\n", os.Args[1])

	if err := http.ListenAndServe(os.Args[1], handlers.LoggingHandler(os.Stderr, r)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
