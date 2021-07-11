package deovr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rafaelmartins/deovr-library/internal/ffmpeg"
	"github.com/rafaelmartins/deovr-library/internal/imagemagick"
)

type VideoSource struct {
	Resolution int    `json:"resolution"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
	URL        string `json:"url"`
}

type Encoding struct {
	Name         string         `json:"name"`
	VideoSources []*VideoSource `json:"videoSources"`
}

type Media struct {
	ID           int         `json:"id,omitempty"`
	Title        string      `json:"title"`
	ThumbnailURL string      `json:"thumbnailUrl"`
	FPS          int         `json:"fps,omitempty"`
	Is3D         bool        `json:"is3d,omitempty"`
	ViewAngle    int         `json:"viewAngle,omitempty"`
	StereoMode   string      `json:"stereoMode,omitempty"`
	VideoLength  int         `json:"videoLength,omitempty"`
	Encodings    []*Encoding `json:"encodings,omitempty"`
	Path         string      `json:"path,omitempty"`
}

type NonMedia struct {
	Title string
	Path  string
}

type Scene struct {
	Name         string      `json:"name"`
	List         []*Media    `json:"list"`
	ListNonMedia []*NonMedia `json:"-"`
	dir          string
}

type DeoVR struct {
	Scenes []*Scene `json:"scenes"`
	mux    sync.Mutex
}

func (d *DeoVR) LoadScene(name string, directory string, host string) error {
	dirAbs, err := filepath.Abs(directory)
	if err != nil {
		return err
	}

	scene := &Scene{
		Name: name,
		dir:  directory,
	}

	if err := filepath.Walk(dirAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path != dirAbs && info.Mode().IsDir() {
			return filepath.SkipDir
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		fileName := filepath.Base(path)
		mtype := mime.TypeByExtension(filepath.Ext(path))
		isVideo := strings.HasPrefix(mtype, "video/")
		isImage := strings.HasPrefix(mtype, "image/")
		if !(isVideo || isImage) {
			nm := &NonMedia{
				Title: fileName,
				Path:  fmt.Sprintf("http://%s/media/%s/%s", host, name, fileName),
			}
			scene.ListNonMedia = append(scene.ListNonMedia, nm)
			return nil
		}

		deovrDir := filepath.Join(filepath.Dir(path), ".deovr")
		if _, err := os.Stat(deovrDir); os.IsNotExist(err) {
			if err := os.MkdirAll(deovrDir, 0777); err != nil {
				return err
			}
		}

		if isImage {
			log.Printf("[%s] Processing image: %s", scene.Name, path)

			thumbPath := filepath.Join(deovrDir, fileName)
			if tinfo, err := os.Stat(thumbPath); os.IsNotExist(err) || info.ModTime().After(tinfo.ModTime()) {
				log.Printf("[%s] Generating image thumbnail: %s", scene.Name, path)
				thumbData, err := imagemagick.GenerateThumbnail(path, nil, 250, 141)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: thumbnail: %s: %s\n", path, err)
					return nil
				}

				if err := ioutil.WriteFile(thumbPath, thumbData, 0666); err != nil {
					return err
				}
			}

			image := &Media{
				Title:        fileName,
				ThumbnailURL: fmt.Sprintf("http://%s/thumb/%s/%s", host, name, fileName),
				Path:         fmt.Sprintf("http://%s/media/%s/%s", host, name, fileName),
			}
			scene.List = append(scene.List, image)
			return nil
		}

		log.Printf("[%s] Processing video: %s", scene.Name, path)
		var videoData *ffmpeg.ProbeVideoData
		dataPath := filepath.Join(deovrDir, fileName+".json")
		if tinfo, err := os.Stat(dataPath); os.IsNotExist(err) || info.ModTime().After(tinfo.ModTime()) {
			log.Printf("[%s] Generating video data: %s", scene.Name, path)
			var err error
			videoData, err = ffmpeg.ProbeVideo(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: data: %s: %s\n", path, err)
				return nil
			}

			f, err := os.Create(dataPath)
			if err != nil {
				return err
			}

			if err := json.NewEncoder(f).Encode(videoData); err != nil {
				return err
			}
		} else {
			f, err := os.Open(dataPath)
			if err != nil {
				return err
			}

			videoData = &ffmpeg.ProbeVideoData{}
			if err := json.NewDecoder(f).Decode(videoData); err != nil {
				return err
			}
		}

		thumbPath := filepath.Join(deovrDir, fileName+".png")
		if tinfo, err := os.Stat(thumbPath); os.IsNotExist(err) || info.ModTime().After(tinfo.ModTime()) {
			log.Printf("[%s] Generating video thumbnail: %s", scene.Name, path)
			snapshot, err := ffmpeg.GenerateVideoSnapshot(path, videoData.Duration/2, 250)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: snapshot: %s: %s\n", path, err)
				return nil
			}

			thumbData, err := imagemagick.GenerateThumbnail("-", snapshot, 250, 141)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: thumbnail: %s: %s\n", path, err)
				return nil
			}

			if err := ioutil.WriteFile(thumbPath, thumbData, 0666); err != nil {
				return err
			}
		}

		video := &Media{
			Title:        fileName,
			FPS:          videoData.FramesPerSecond,
			VideoLength:  videoData.Duration,
			ThumbnailURL: fmt.Sprintf("http://%s/thumb/%s/%s.png", host, name, fileName),
			Encodings: []*Encoding{
				{
					Name: videoData.CodecName,
					VideoSources: []*VideoSource{
						{
							Resolution: videoData.Height,
							Height:     videoData.Height,
							Width:      videoData.Width,
							URL:        fmt.Sprintf("http://%s/media/%s/%s", host, name, fileName),
						},
					},
				},
			},
		}

		// silly heuristics to detect 3d mode
		a180 := strings.Contains(fileName, "_180")
		a200 := strings.Contains(fileName, "_MKX200")
		a360 := strings.Contains(fileName, "_360")
		h := strings.Contains(fileName, "_3dh")
		v := strings.Contains(fileName, "_3dv")
		sbs := strings.Contains(fileName, "_SBS")
		lr := strings.Contains(fileName, "_LR")
		tb := strings.Contains(fileName, "_TB")
		ou := strings.Contains(fileName, "_OverUnder")
		if a180 || a200 || a360 || h || v || sbs || lr || tb || ou {
			video.Is3D = true
			video.ViewAngle = 180
			video.StereoMode = "sbs"
			if a200 {
				video.ViewAngle = 200
			} else if a360 {
				video.ViewAngle = 360
			}
			if v || tb || ou {
				video.StereoMode = "tb"
			}
		}
		if !video.Is3D {
			if videoData.ScreenRatio > (16.0 / 9.0) {
				video.Is3D = true
				video.ViewAngle = 180
				video.StereoMode = "sbs"
			}
		}

		scene.List = append(scene.List, video)

		return nil
	}); err != nil {
		return err
	}

	d.mux.Lock()
	d.Scenes = append(d.Scenes, scene)
	d.mux.Unlock()

	return nil
}

func (d *DeoVR) GetSceneByName(sceneName string) *Scene {
	for _, scene := range d.Scenes {
		if scene.Name == sceneName {
			return scene
		}
	}
	return nil
}

func (d *DeoVR) getSceneDirectory(sceneName string) string {
	if s := d.GetSceneByName(sceneName); s != nil {
		return s.dir
	}
	return ""
}

func (d *DeoVR) GetMediaPath(sceneName string, fileName string) (string, error) {
	dir := d.getSceneDirectory(sceneName)
	f := filepath.Join(dir, fileName)
	if _, err := os.Stat(f); err != nil {
		return "", err
	}
	return f, nil
}

func (d *DeoVR) GetThumbnailPath(sceneName string, fileName string) (string, error) {
	dir := d.getSceneDirectory(sceneName)
	f := filepath.Join(dir, ".deovr", fileName)
	if _, err := os.Stat(f); err != nil {
		return "", err
	}
	return f, nil
}
