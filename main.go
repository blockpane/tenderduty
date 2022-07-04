package main

import (
	_ "embed"
	"flag"
	"fmt"
	td2 "github.com/blockpane/tenderduty/v2/td2"
	"golang.org/x/term"
	"log"
	"os"
	"syscall"
)

//go:embed example-config.yml
var defaultConfig []byte

func main() {
	var configFile, encryptedFile, stateFile, password string
	var dumpConfig, encryptConfig, decryptConfig bool
	flag.StringVar(&configFile, "f", "config.yml", "configuration file to use")
	flag.StringVar(&encryptedFile, "encrypted-config", "config.yml.asc", "encrypted config file, only valid with -encrypt or -decrypt flag")
	flag.StringVar(&stateFile, "state", ".tenderduty-state.json", "file for storing state between restarts")
	flag.StringVar(&password, "password", "", "password to use for encrypting/decrypting the config, if unset will prompt, also can use ENV var 'PASSWORD'")
	flag.BoolVar(&dumpConfig, "example-config", false, "print the an example config.yml and exit")
	flag.BoolVar(&encryptConfig, "encrypt", false, "encrypt the file specified by -f to -encrypted-config")
	flag.BoolVar(&decryptConfig, "decrypt", false, "decrypt the file specified by -encrypted-config to -f")
	flag.Parse()

	if dumpConfig {
		fmt.Println(string(defaultConfig))
		os.Exit(0)
	}

	if encryptConfig || decryptConfig {
		if password == "" {
			if os.Getenv("PASSWORD") != "" {
				password = os.Getenv("PASSWORD")
			} else {
				fmt.Print("Please enter the encryption password: ")
				pass, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					log.Fatal(err)
				}
				password = string(pass)
				pass = nil
			}
		}
		var e error
		if encryptConfig {
			e = td2.EncryptedConfig(configFile, encryptedFile, password, false)
		} else {
			e = td2.EncryptedConfig(configFile, encryptedFile, password, true)
		}
		if e != nil {
			log.Fatalln(e)
		}
		os.Exit(0)
	}

	err := td2.Run(configFile, stateFile, &password)
	if err != nil {
		log.Println(err.Error(), "... exiting.")
	}
}
