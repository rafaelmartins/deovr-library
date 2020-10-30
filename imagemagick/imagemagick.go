package imagemagick

import (
	"fmt"
	"os/exec"
)

func GenerateThumbnail(imagePath string, imageData []byte, width int, height int) ([]byte, error) {
	cmd := exec.Command(
		"convert",
		imagePath,
		"-resize", fmt.Sprintf("%dx%d", width, height),
		"-gravity", "center",
		"-background", "black",
		"-extent", fmt.Sprintf("%dx%d", width, height),
		"-",
	)

	if imagePath == "-" && len(imageData) > 0 {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}

		go func() {
			defer stdin.Close()
			stdin.Write(imageData)
		}()
	}

	return cmd.Output()
}
