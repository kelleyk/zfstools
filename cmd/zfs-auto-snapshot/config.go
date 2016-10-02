package main

import (
	"fmt"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type seriesConfig struct {
	Label    string
	Interval time.Duration
	Keep     int
}

type configFile struct {
	Series []*seriesConfig
	Foo    string
}

func loadConfig(path string) (*configFile, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &configFile{}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *configFile) Validate() error {
	for _, series := range c.Series {
		if series.Label == "" {
			return fmt.Errorf("series has empty label")
		}
		if series.Keep <= 0 {
			return fmt.Errorf("series has keep <= 0")
		}
		if series.Interval <= time.Duration(0) {
			return fmt.Errorf("series has interval <= 0")
		}
	}

	return nil
}
