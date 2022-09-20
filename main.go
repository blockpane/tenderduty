package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	td2 "github.com/blockpane/tenderduty/v2/td2"
)

//go:embed example-config.yml
var defaultConfig []byte

func main() {
	var configFile, chainConfigDirectory, stateFile string
	var dumpConfig bool
	flag.StringVar(&configFile, "f", "config.yml", "configuration file to use")
	flag.StringVar(&stateFile, "state", ".tenderduty-state.json", "file for storing state between restarts")
	flag.StringVar(&chainConfigDirectory, "cc", "chains.d", "directory containing additional chain specific configurations")
	flag.BoolVar(&dumpConfig, "example-config", false, "print the an example config.yml and exit")
	flag.Parse()

	if dumpConfig {
		fmt.Println(string(defaultConfig))
		os.Exit(0)
	}

	err := td2.Run(configFile, stateFile, chainConfigDirectory)
	if err != nil {
		log.Println(err.Error(), "... exiting.")
	}
}
