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
	"github.com/rafaelmartins/deovr-library/deovr"
)

var data = &deovr.DeoVR{}

func DeoVRHandler(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func VideoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	f, err := data.GetVideoPath(vars["scene"], vars["file"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "file not found\n")
		return
	}
	http.ServeFile(w, r, f)
}

func ThumbHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	f, err := data.GetVideoThumbnailPath(vars["scene"], vars["file"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "file not found\n")
		return
	}
	http.ServeFile(w, r, f)
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
	r.HandleFunc("/deovr", DeoVRHandler)
	r.HandleFunc("/video/{scene}/{file}", VideoHandler)
	r.HandleFunc("/thumb/{scene}/{file}", ThumbHandler)

	fmt.Fprintf(os.Stderr, "\n * Running on http://%s/deovr\n\n", os.Args[1])

	if err := http.ListenAndServe(os.Args[1], handlers.LoggingHandler(os.Stderr, r)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
