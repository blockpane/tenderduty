# TenderDuty v2

[![Go Reference](https://pkg.go.dev/badge/github.com/blockpane/tenderduty.svg)](https://pkg.go.dev/github.com/blockpane/tenderduty)
[![Gosec](https://github.com/blockpane/tenderduty/workflows/Gosec/badge.svg)](https://github.com/blockpane/tenderduty/actions?query=workflow%3AGosec)
[![CodeQL](https://github.com/blockpane/tenderduty/workflows/CodeQL/badge.svg)](https://github.com/blockpane/tenderduty/actions?query=workflow%3ACodeQL)

![dashboard screenshot](docs/dash.png)
TenderDuty v2 is complete rewrite of the original tenderduty graciously sponsored by the [Osmosis Grants Program](https://grants.osmosis.zone/). This new version adds a web dashboard, prometheus exporter, telegram and discord notifications, multi-chain support, more granular alerting, and more types of alerts.

Documentation will be provided soon. The example-config.yml file is well-commented.

30 second quickstart for beta testers:


if you'd prefer to containerize and not build locally, you can:

```
$ mkdir tenderduty && cd tenderduty
$ docker run --rm ghcr.io/blockpane/tenderduty:release-v2 -example-config >config.yml
# edit config.yml and add chains, notification methods etc.
$ docker run -d --name tenderduty -p "8888:8888" --restart unless-stopped -v $(pwd)/config.yml:/var/lib/tenderduty/config.yml ghcr.io/blockpane/tenderduty:release-v2
$ docker logs -f --tail 20 tenderduty
```

Or if building from source:

```
$ git clone https://github.com/blockpane/tenderduty
$ cd tenderduty
$ git checkout release/v2
$ cp example-config.yml config.yml
# edit config.yml
$ go get ./...
$ go install
$ ~/go/bin/tenderduty
```
