package main

import (
	"flag"
	tenderduty "github.com/blockpane/tenderduty"
	"log"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "f", "config.yml", "configuration file to use")
	flag.Parse()

	err := tenderduty.Run(configFile)
	if err != nil {
		log.Println(err.Error(), "... exiting.")
	}
}
