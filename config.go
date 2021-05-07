package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

var configuration Configuration

var configPath = "config.toml"

func init() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	configuration, err = loadConfig(exPath + "/" + configPath)
	if err != nil {
		configuration, err = loadConfig(configPath)
		if err != nil {
			panic(err)
		}
	} else {
		configPath = exPath + "/" + configPath
	}
}

type Main struct {
	ReadEntireLog         bool `comment:"Should we read the entire character log or just new entries"`
	LogPollRate           int  `comment:"How often to check for new entries in the character log in seconds"`
	RemainingTicksWarning int  `comment:"How many remaining ticks to issue a warning at"`
	UseTTS                bool `comment:"Use TTS to announce spells on last tick"`
	ShowUnknown           bool `comment:"Shows determinal spells before knowing their target"`
}

type SpellOverride struct {
	Name    string `comment:"Name of spell to override"`
	SpellID int    `comment:"ID of overrode spell"`
}

type Everquest struct {
	ExtendDotsPercent float64 `comment:"How long to extend dots due to focus 1.2 == timeburn"`
	LogPath           string  `comment:"path to character log file"`
	PlayerClass       string  `comment:"Class of character: Necromancer"`
	SpellDB           string  `comment:"path to the lucydb spell database"`
	TakenDamageRegex  string  `comment:"Regex to find when a mob takes damage from your dot"`
	BeginCastingRegex string  `comment:"Regex to find when you cast a spell"`
	InterruptedRegex  string  `comment:"Regex to find when your spell is interrupted"`
	FizzledRegex      string  `comment:"Regex to find when you fizzle a spell"`
	WornOffRegex      string  `comment:"Regex to find when your spell wears off"`
	OverwrittenRegex  string  `comment:"Regex to find when your spell is overwritten"`
	KilledRegex       string  `comment:"Regex to find when the npc is killed"`
	DiedRegex         string  `comment:"Regex to find when you have died, removing all monitored dots"`
	ResistedRegex     string  `comment:"Regex to find when your spell is resisted"`
	DetrimentalHaste  float64 `comment:"How much spell haste for determinals, AH4 = 1.33"`
}

type Log struct {
	Level int    `comment:"How much to log Warn:0 Err:1 Info:2 Debug:3"`
	Path  string `comment:"Where to store the log file use linux formatting or escape slashes for windows"`
}

type Configuration struct {
	Main      Main
	Everquest Everquest
	Log       Log
	Overrides []SpellOverride `comment:"Spell that finds as wrong ID, force an ID here"`
}

func loadConfig(path string) (Configuration, error) {
	config := Configuration{}
	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = toml.Unmarshal(configFile, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func (c Configuration) save(path string) {
	out, err := toml.Marshal(c)
	if err != nil {
		Err.Printf("Error marshalling config: %s", err.Error())
	}
	err = ioutil.WriteFile(path, out, 0644)
	if err != nil {
		Err.Printf("Error writing config: %s", err.Error())
	}
}
