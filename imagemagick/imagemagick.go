package imagemagick

import (
	"fmt"
	"os/exec"
)

func GenerateImageThumbnail(imagePath string, width int, height int) ([]byte, error) {
	cmd := exec.Command(
		"convert",
		imagePath,
		"-resize", fmt.Sprintf("%dx%d", width, height),
		"-",
	)
	return cmd.Output()
}
