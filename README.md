# TenderDuty v2

**This is pre-alpha software. Development is taking place very rapidly. It's not ready for production use, 
and is not yet a viable replacement for the original Tenderduty.**

![dashboard screenshot](docs/dash.png)
TenderDuty v2 is complete rewrite of the original tenderduty graciously sponsored by the [Osmosis Grants Program](https://grants.osmosis.zone/). This new version adds a web dashboard, prometheus exporter, telegram and discord notifications, multi-chain support, more granular alerting, and more types of alerts.

Some warnings: 

* Not all alerts types are working.
* Alert state isn't saved between restarts
* No rate limiting is in place, so you might get 10^24 alert notifications if something goes wrong.
* Essentially no testing has been done
* No docs exist, and won't until it's ready. If you want to play with this it's up to you to figure out how to configure it.

30 second quickstart for alpha testers:

```
git clone https://github.com/blockpane/tenderduty
cd tenderduty
git checkout feature/v2
cp example-config.yml config.yml
# edit config.yml
go get ./...
go run cmd/tenderduty/main.go
```