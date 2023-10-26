package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Scene struct {
	Identifier string
	Path       string
}

type Scenes []*Scene

func (s *Scenes) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.AliasNode {
		value = value.Alias
	}

	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("config: scenes: not a mapping (line %d, column %d)", value.Line, value.Column)
	}

	identifier := ""
	for i, cnt := range value.Content {
		if i%2 == 0 {
			if err := cnt.Decode(&identifier); err != nil {
				return err
			}
		} else {
			scene := &Scene{
				Identifier: identifier,
			}

			if cnt.Kind == yaml.AliasNode {
				cnt = cnt.Alias
			}

			if err := cnt.Decode(&scene.Path); err != nil {
				return err
			}

			*s = append(*s, scene)
		}
	}
	return nil
}
