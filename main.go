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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := gallery.Index(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sceneHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if s := data.GetSceneByName(vars["scene"]); s == nil {
		http.NotFound(w, r)
	} else if err := gallery.Scene(w, s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deoVRHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mediaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if f, err := data.GetMediaPath(vars["scene"], vars["file"]); err != nil {
		http.NotFound(w, r)
	} else {
		http.ServeFile(w, r, f)
	}
}

func thumbHandler(w http.ResponseWriter, r *http.Request) {
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

	addr := os.Args[1]
	pieces := strings.Split(addr, ":")
	if pieces[0] == "" {
		fmt.Fprintf(os.Stderr, "Error: hostname required for URL generation\n\n")
		usage()
	}

	if len(pieces) == 1 {
		addr += ":80"
	} else if _, err := strconv.Atoi(pieces[1]); err != nil {
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
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/deovr", deoVRHandler)
	r.HandleFunc("/scene/{scene}", sceneHandler)
	r.HandleFunc("/media/{scene}/{file}", mediaHandler)
	r.HandleFunc("/thumb/{scene}/{file}", thumbHandler)

	fmt.Fprintf(os.Stderr, "\n * Running on http://%s/\n\n", os.Args[1])

	if err := http.ListenAndServe(addr, handlers.LoggingHandler(os.Stderr, r)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
