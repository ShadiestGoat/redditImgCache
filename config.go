package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/shadiestgoat/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
		ExposeSubs bool `yaml:"advertizeSubreddit"`
		DB string `yaml:"dbURL"`
	} `yaml:"server"`
	Logger struct {
		Discord *struct {
			Prefix string `yaml:"prefix"`
			Webhook string `yaml:"webhook"`
		} `yaml:"discord"`
		File *struct {
			Name string `yaml:"name"`
			MaxFiles int `yaml:"maxFiles"`
			NewestAt0 bool `yaml:"newestAt0"`
		} `yaml:"file"`
	} `yaml:"logger"`
	Subs map[string]ConfigSub `yaml:"subs"`
}

type ConfigSub struct {
	Hydrate int `yaml:"hydrate"`
	SaveNSFW *bool `yaml:"saveNSFW"`
	Alias string `yaml:"alias"`
}

var conf = &Config{}

func panicBeforeLog(msg string) {
	log.Init(log.NewLoggerPrint())
	log.Fatal(msg)
}

func init() {
	b, err := os.ReadFile("config.yaml")
	if err != nil {
		panicBeforeLog(fmt.Sprintf("Couldn't open config file: %v", err))
	}

	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		panicBeforeLog(fmt.Sprintf("Couldn't load config file: %v", err))
	}
}

func defaultUInt(v *int, name string, defVal int) {
	if *v <= 0 {
		log.Warn("%v invalid - setting default value of %v", name, defVal)
		*v = defVal
	}
}

func init() {
	loggers := []log.LogCB{
		log.NewLoggerPrint(),
	}

	if conf.Logger.Discord != nil && conf.Logger.Discord.Webhook != "" {
		_, err := url.Parse(conf.Logger.Discord.Webhook)
		if err != nil {
			panicBeforeLog("Could not load discord parser: invalid webhook url!")
		}

		c := conf.Logger.Discord

		c.Prefix = strings.TrimSpace(c.Prefix)
		loggers = append(loggers, log.NewLoggerDiscordWebhook(c.Prefix, c.Webhook))
	}

	if conf.Logger.File != nil && conf.Logger.File.Name != "" {
		c := conf.Logger.File
		m := log.FILE_OVERWRITE
		if c.MaxFiles != 1 {
			if c.NewestAt0 {
				m = log.FILE_DESCENDING
			} else {
				m = log.FILE_ASCENDING
			}
		}

		loggers = append(loggers, log.NewLoggerFileComplex(c.Name, m, c.MaxFiles))
	}

	log.Init(loggers...)

	defaultUInt(&conf.Server.Port, "Port", 3000)

	for k, v := range conf.Subs {
		if v.Hydrate <= 0 {
			log.Warn("Subreddit '%v' doesn't have a valid hydration time - setting default (24h)", k)
			v.Hydrate = 24
		}
		if v.SaveNSFW == nil {
			v.SaveNSFW = new(bool)
			*v.SaveNSFW = true
		}
		if v.Alias == "" {
			v.Alias = k
		}

		conf.Subs[k] = v
	}
}