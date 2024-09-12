package main

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/shadiestgoat/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port       int    `yaml:"port"`
		ExposeSubs bool   `yaml:"advertizeSubreddit"`
		DB         string `yaml:"dbURL"`
		RefreshPad int    `yaml:"refreshPad"`
	} `yaml:"server"`
	HttpStuff struct {
		UserAgent   string `yaml:"userAgent"`
		Credentials string `yaml:"credentials"`
	} `yaml:"httpStuff"`
	Logger struct {
		Discord *struct {
			Prefix  string `yaml:"prefix"`
			Webhook string `yaml:"webhook"`
		} `yaml:"discord"`
		File *struct {
			Name      string `yaml:"name"`
			MaxFiles  int    `yaml:"maxFiles"`
			NewestAt0 bool   `yaml:"newestAt0"`
		} `yaml:"file"`
	} `yaml:"logger"`
	Subs map[string]ConfigSub `yaml:"subs"`
}

type ConfigSub struct {
	Hydrate  int    `yaml:"hydrate"`
	SaveNSFW *bool  `yaml:"saveNSFW"`
	Alias    string `yaml:"alias"`
}

var ReEnvVar = regexp.MustCompile(`\$([A-Z_]*[A-Z])`)

func LoadEnvConfig(s *string) {
	matches := ReEnvVar.FindAllStringSubmatch(*s, -1)

	replacers := []string{}

	for _, m := range matches {
		env_var := m[1]
		v := os.Getenv(env_var)

		if v == "" {
			continue
		}

		replacers = append(replacers, m[0], v)
	}

	*s = strings.NewReplacer(replacers...).Replace(*s)
}

var conf = &Config{}

func panicBeforeLog(msg string) {
	log.Init(log.NewLoggerPrint())
	log.Fatal(msg)
}

func ReadConfig() {
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
		log.PrintWarn("%v invalid - setting default value of %v", name, defVal)
		*v = defVal
	}
}

var Aliases = map[string]bool{}

func ValidateConfig() {
	loggers := []log.LogCB{
		log.NewLoggerPrint(),
	}

	LoadEnvConfig(&conf.Server.DB)

	if conf.Logger.Discord != nil && conf.Logger.Discord.Webhook != "" {
		LoadEnvConfig(&conf.Logger.Discord.Webhook)

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

	conf.HttpStuff.UserAgent = strings.TrimSpace(conf.HttpStuff.UserAgent)

	if conf.HttpStuff.UserAgent == "" {
		conf.HttpStuff.UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0"
	}

	if conf.HttpStuff.Credentials != "" {
		if !strings.Contains(conf.HttpStuff.Credentials, ":") {
			LoadEnvConfig(&conf.HttpStuff.Credentials)
		
			log.Fatal("Your reddit credentials need to be in the form of 'client_id:client_secret'!")
			return
		}

		conf.HttpStuff.Credentials = "Basic " + base64.StdEncoding.EncodeToString([]byte(conf.HttpStuff.Credentials))
		log.Debug("Credential detected & loaded!")
	}

	for k, v := range conf.Subs {
		if v.Hydrate <= 0 {
			log.PrintWarn("Subreddit '%v' doesn't have a valid hydration time - setting default (24h)", k)
			v.Hydrate = 24
		}
		if v.SaveNSFW == nil {
			v.SaveNSFW = new(bool)
			*v.SaveNSFW = true
		}
		if v.Alias == "" {
			v.Alias = k
		}

		Aliases[v.Alias] = true

		conf.Subs[k] = v
	}
}

func LoadFullConfig() {
	ReadConfig()
	ValidateConfig()	
}