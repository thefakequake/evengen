package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	defaultOutDir     = "out"
	defaultIgnoreDirs = []string{
		"rich_presence",
		"tutorials",
		"game_sdk",
		"policies_and_agreements",
		"game_and_server_management",
		"tutorials",
		"dispatch",
	}
	defaultPackage = "eventide"
)

type Config struct {
	Token      string   `json:"token"`
	OutDir     string   `json:"outDir"`
	IgnoreDirs []string `json:"ignoreDirs"`
	Package    string   `json:"package"`
}

func GetConfig() (*Config, error) {
	var c Config
	var err error

	f, err := os.Open(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return &c, err
		}
		c.OutDir = defaultOutDir
		c.IgnoreDirs = defaultIgnoreDirs
		c.Package = defaultPackage

		dat, err := json.MarshalIndent(&c, "", "    ")
		if err != nil {
			return &c, err
		}

		err = os.WriteFile(configFile, dat, 0)
		if err != nil {
			return &c, err
		}

		fmt.Printf("created new config file %s\n", configFile)
		return &c, nil
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&c)
	return &c, err
}
