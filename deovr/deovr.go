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

	"github.com/rafaelmartins/deovr-library/ffmpeg"
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

type Video struct {
	ID           int         `json:"id,omitempty"`
	Title        string      `json:"title"`
	FPS          int         `json:"fps"`
	Is3D         bool        `json:"is3d"`
	ViewAngle    int         `json:"viewAngle,omitempty"`
	StereoMode   string      `json:"stereoMode,omitempty"`
	VideoLength  int         `json:"videoLength"`
	ThumbnailURL string      `json:"thumbnailUrl"`
	Encodings    []*Encoding `json:"encodings"`
}

type Scene struct {
	Name string   `json:"name"`
	List []*Video `json:"list"`
	dir  string
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

		if !strings.HasPrefix(mime.TypeByExtension(filepath.Ext(path)), "video/") {
			return nil
		}

		fileName := filepath.Base(path)
		deovrDir := filepath.Join(filepath.Dir(path), ".deovr")
		if _, err := os.Stat(deovrDir); os.IsNotExist(err) {
			if err := os.MkdirAll(deovrDir, 0777); err != nil {
				return err
			}
		}

		log.Printf("[%s] Processing video: %s", scene.Name, path)

		var videoData *ffmpeg.ProbeVideoData
		dataPath := filepath.Join(deovrDir, fileName+".json")
		if tinfo, err := os.Stat(dataPath); os.IsNotExist(err) || info.ModTime().After(tinfo.ModTime()) {
			log.Printf("[%s] Generating video data: %s", scene.Name, path)
			var err error
			videoData, err = ffmpeg.ProbeVideo(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s: %s\n", path, err)
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
			thumbData, err := ffmpeg.GenerateVideoThumbnail(path, videoData.Duration/2, int(250.0*videoData.ScreenRatio), 250)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s: %s\n", path, err)
				return nil
			}

			if err := ioutil.WriteFile(thumbPath, thumbData, 0666); err != nil {
				return err
			}
		}

		video := &Video{
			Title:        fileName,
			FPS:          videoData.FramesPerSecond,
			VideoLength:  videoData.Duration,
			ThumbnailURL: fmt.Sprintf("http://%s/thumb/%s/%s.png", host, name, fileName),
			Encodings: []*Encoding{
				&Encoding{
					Name: videoData.CodecName,
					VideoSources: []*VideoSource{
						&VideoSource{
							Resolution: videoData.Height,
							Height:     videoData.Height,
							Width:      videoData.Width,
							URL:        fmt.Sprintf("http://%s/video/%s/%s", host, name, filepath.Base(path)),
						},
					},
				},
			},
		}

		if videoData.ScreenRatio > (16.0 / 9.0) {
			video.Is3D = true
			video.ViewAngle = 180
			video.StereoMode = "sbs"
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

func (d *DeoVR) GetVideoPath(sceneName string, fileName string) (string, error) {
	dir := d.getSceneDirectory(sceneName)
	f := filepath.Join(dir, fileName)
	if _, err := os.Stat(f); err != nil {
		return "", err
	}
	return f, nil
}

func (d *DeoVR) GetVideoThumbnailPath(sceneName string, fileName string) (string, error) {
	dir := d.getSceneDirectory(sceneName)
	f := filepath.Join(dir, ".deovr", fileName)
	if _, err := os.Stat(f); err != nil {
		return "", err
	}
	return f, nil
}
