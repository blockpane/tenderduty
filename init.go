package tenderduty

import (
	"embed"
	"fmt"
	dash "github.com/blockpane/tenderduty/dashboard"
	"log"
	"os"
	"strings"
	"time"
)

//go:embed static/*
var content embed.FS

func init() {
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stderr)
	dash.Content = content

	// use a channel for logging, two reasons: several logs could hit at once (formatting,) and to broadcast
	// messages to the monitoring dashboard
	go func() {
		for msg := range logs {
			msg = strings.TrimRight(strings.TrimLeft(fmt.Sprint(msg), "["), "]")
			log.Println("tenderduty | ", msg)
			if td.EnableDash && !td.HideLogs && td.logChan != nil {
				td.logChan <- dash.LogMessage{
					MsgType: "log",
					Ts:      time.Now().UTC().Unix(),
					Msg:     msg.(string),
				}
			}
		}
	}()
}

var logs = make(chan interface{})

func l(v ...any) {
	logs <- v
}
