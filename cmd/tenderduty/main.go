package main

import (
	"flag"
	tenderduty "github.com/blockpane/tenderduty"
	"log"
)

func main() {
	var configFile, stateFile string
	var dumpConfig bool
	flag.StringVar(&configFile, "f", "config.yml", "configuration file to use")
	flag.StringVar(&stateFile, "state", ".tenderduty-state.json", "file for storing state between restarts")
	flag.BoolVar(&dumpConfig, "example-config", false, "print the an example config.yml and exit")
	flag.Parse()

	err := tenderduty.Run(configFile, stateFile, dumpConfig)
	if err != nil {
		log.Println(err.Error(), "... exiting.")
	}
}
