package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

func ReadYml[T any](name string) (T, error) {
	var data T
	content, err := os.ReadFile(name)
	if err != nil {
		return data, err
	}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return data, err
	}
	return data, nil
}
