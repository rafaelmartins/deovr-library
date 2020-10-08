package gallery

import (
	"errors"
	"html/template"
	"io"

	"github.com/rafaelmartins/deovr-library/deovr"
)

func Index(w io.Writer, d *deovr.DeoVR) error {
	if d == nil {
		return errors.New("DeoVR is nil")
	}

	t, err := template.New("index").Parse(`
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
	t, err := template.New("scene").Parse(`
<html>
<body>
<h1>Scene: {{.Name}}</h1>
<ul>
{{- range .List}}
<li><a href="{{(index (index .Encodings 0).VideoSources 0).URL}}">{{.Title}}</a></li>
{{- end}}
</ul>
</body>
</html>
`)
	if err != nil {
		return err
	}

	return t.Execute(w, s)
}
