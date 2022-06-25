package tenderduty

import (
	"fmt"
	dash "github.com/blockpane/tenderduty/dashboard"
	"log"
	"time"
)

var td = &Config{}

func Run(configFile string) error {
	var err error
	td, err = loadConfig(configFile)
	if err != nil {
		return err
	}
	fatal, problems := validateConfig(td)
	for _, p := range problems {
		fmt.Println(p)
	}
	if fatal {
		log.Fatal("tenderduty the configuration is invalid, refusing to start")
	}
	log.Println("tenderduty config is valid, starting tenderduty with", len(td.Chains), "chains")

	defer td.cancel()

	go func() {
		for {
			select {
			case alert := <-td.alertChan:
				go func(msg *alertMsg) {
					var e error
					e = notifyDiscord(msg)
					if e != nil {
						l(msg.chain, "error sending alert to discord", e.Error())
					}
					e = notifyTg(msg)
					if e != nil {
						l(msg.chain, "error sending alert to telegram", e.Error())
					}
					e = notifyPagerduty(msg)
					if e != nil {
						l(msg.chain, "error sending alert to pagerduty", e.Error())
					}
				}(alert)
			case <-td.ctx.Done():
				return
			}
		}
	}()

	if td.EnableDash {
		l("starting dashboard on", td.Listen)
		go dash.Serve(td.Listen, td.updateChan, td.logChan, td.HideLogs)
	}
	if td.Prom {
		go prometheusExporter(td.ctx, td.statsChan)
	}

	for k := range td.Chains {
		cc := td.Chains[k]

		go func(cc *ChainConfig, name string) {
			// alert worker
			go cc.watch()

			// node health checks:
			go func() {
				for {
					cc.monitorHealth(td.ctx, name)
					time.Sleep(time.Minute)
				}
			}()

			// websocket subscription and occasional validator info refreshes
			for {
				e := cc.newRpc()
				if e != nil {
					l(cc.ChainId, e)
					td.alert(cc.name, e.Error(), "critical", false, &cc.name)
					time.Sleep(5 * time.Second)
					continue
				}
				e = cc.GetValInfo(true)
				if e != nil {
					l("🛑", cc.ChainId, e)
				}
				cc.WsRun()
				l(cc.ChainId, "🌀 not working! Will restart monitoring in 1 minute")
				time.Sleep(time.Minute)
			}
		}(cc, k)
	}
	<-td.ctx.Done()

	return err
}
