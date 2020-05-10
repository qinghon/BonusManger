package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Remarks string `json:"remarks" yaml:"remarks"`
}

const projectPath = "/opt/BonusManger"
const configPath = projectPath + "/config.yaml"

func (c *Config) save() error {
	by, err := yaml.Marshal(c)
	if err != nil {
		log.Error(err)
	}
	return ioutil.WriteFile(configPath, by, 0644)
}
func (c *Config) get() error {
	by, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Error(err)
		return err
	}
	err = yaml.Unmarshal(by, c)
	return err
}
