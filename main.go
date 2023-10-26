package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rafaelmartins/deovr-library/internal/config"
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

func zipHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	s := data.GetSceneByName(vars["scene"])
	if s == nil {
		http.NotFound(w, r)
		return
	}

	if len(s.ListNonMedia) == 0 {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, vars["scene"]))
	if err := s.WriteNonMediaZip(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deoVRHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
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

func check(err any) {
	if err != nil {
		log.Fatal("error: ", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: deovr-library CONFIG_FILE\n")
		os.Exit(1)
	}

	cfg, err := config.Load(os.Args[1])
	check(err)

	for _, scene := range cfg.Scenes {
		if err := data.LoadScene(scene.Identifier, scene.Path, cfg.BaseURL); err != nil {
			check(fmt.Errorf("failed to load scene: %s", err))
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/deovr", deoVRHandler)
	r.HandleFunc("/scene/{scene}.zip", zipHandler)
	r.HandleFunc("/scene/{scene}", sceneHandler)
	r.HandleFunc("/file/{scene}/{file}", fileHandler)
	r.HandleFunc("/thumb/{scene}/{file}", thumbHandler)

	fmt.Fprintf(os.Stderr, "\n * Running on %s\n\n", cfg.BaseURL)
	check(http.ListenAndServe(cfg.Addr, handlers.LoggingHandler(os.Stderr, r)))
}
