package ffmpeg

import (
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type ProbeVideoData struct {
	CodecName       string
	Width           int
	Height          int
	ScreenRatio     float64
	FramesPerSecond int
	Duration        int
}

type ffprobeStream struct {
	CodecName    string `json:"codec_name"`
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobe struct {
	Streams []*ffprobeStream `json:"streams"`
	Format  *ffprobeFormat   `json:"format"`
}

func ProbeVideo(videoPath string) (*ProbeVideoData, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var fp ffprobe
	if err := json.NewDecoder(stdout).Decode(&fp); err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	var videoStream *ffprobeStream
	for _, v := range fp.Streams {
		if v.CodecType == "video" {
			videoStream = v
			break
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("ffprobe: no video stream found: %s", videoPath)
	}

	fps := 0
	p := strings.SplitN(videoStream.AvgFrameRate, "/", 2)
	if len(p) == 2 {
		p0, err := strconv.Atoi(p[0])
		if err != nil {
			return nil, err
		}
		p1, err := strconv.Atoi(p[1])
		if err != nil {
			return nil, err
		}

		fps = int(math.Ceil(float64(p0) / float64(p1)))
	} else {
		var err error
		fps, err = strconv.Atoi(videoStream.AvgFrameRate)
		if err != nil {
			return nil, err
		}
	}

	duration, err := strconv.ParseFloat(fp.Format.Duration, 64)
	if err != nil {
		return nil, err
	}

	return &ProbeVideoData{
		CodecName:       videoStream.CodecName,
		Width:           videoStream.Width,
		Height:          videoStream.Height,
		ScreenRatio:     float64(videoStream.Width) / float64(videoStream.Height),
		FramesPerSecond: fps,
		Duration:        int(math.Ceil(duration)),
	}, nil
}

func GenerateVideoThumbnail(videoPath string, time int, width int, height int) ([]byte, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-ss", strconv.Itoa(time),
		"-i", videoPath,
		"-vf", "thumbnail",
		"-frames:v", "1",
		"-f", "image2pipe",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-c:v", "png",
		"pipe:1",
	)
	return cmd.Output()
}
