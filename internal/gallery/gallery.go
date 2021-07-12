package gallery

import (
	"errors"
	"html/template"
	"io"

	"github.com/rafaelmartins/deovr-library/internal/deovr"
)

func Index(w io.Writer, d *deovr.DeoVR) error {
	if d == nil {
		return errors.New("DeoVR is nil")
	}

	t, err := template.New("index").Parse(`<!DOCTYPE html>
<html>
<body>
<h1>Scenes</h1>
<ul>
{{- range .Scenes}}
<li><a href="/scene/{{.Name}}">{{.Name}}</a></li>
{{- end}}
</ul>
</body>
</html>
`)
	if err != nil {
		return err
	}

	return t.Execute(w, d)
}

func Scene(w io.Writer, s *deovr.Scene) error {
	if s == nil {
		return errors.New("Scene is nil")
	}

	// FIXME: show thumbnails
	t, err := template.New("scene").Parse(`<!DOCTYPE html>
<html>
<body>
<h1>Scene: {{.Name}}</h1>
<ul>
{{- range .List}}
{{- if .Encodings}}
<li>
<a href="{{(index (index .Encodings 0).VideoSources 0).URL}}">
{{- if .ThumbnailURL}}
<img src="{{.ThumbnailURL}}">
{{- end}}
{{.Title}}
</a>
</li>
{{- else}}
<li>
<a href="{{.Path}}">
{{- if .ThumbnailURL}}
<img src="{{.ThumbnailURL}}">
{{- end}}
{{.Title}}
</a>
</li>
{{- end}}
{{- end}}
</ul>
{{- if .ListNonMedia}}
<h2>Non-media files (<a href="{{.ZipNonMediaURL}}">ZIP</a>)</h2>
<ul>
{{- range .ListNonMedia}}
<li>
<a href="{{.Path}}">{{.Title}}</a>
</li>
{{- end}}
</ul>
{{- end}}
</body>
</html>
`)
	if err != nil {
		return err
	}

	return t.Execute(w, s)
}
