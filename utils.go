package main

import (
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

func LoadFromYaml(path string, cfg interface{}) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, cfg)
}
