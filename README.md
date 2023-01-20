# TenderDuty v2

[![Go Reference](https://pkg.go.dev/badge/github.com/blockpane/tenderduty.svg)](https://pkg.go.dev/github.com/blockpane/tenderduty)
[![Gosec](https://github.com/blockpane/tenderduty/workflows/Gosec/badge.svg)](https://github.com/blockpane/tenderduty/actions?query=workflow%3AGosec)
[![CodeQL](https://github.com/blockpane/tenderduty/workflows/CodeQL/badge.svg)](https://github.com/blockpane/tenderduty/actions?query=workflow%3ACodeQL)

Tenderduty is a comprehensive monitoring tool for Tendermint chains. Its primary function is to alert a validator if they are missing blocks, and has many other features.

v2 is complete rewrite of the original tenderduty graciously sponsored by the [Osmosis Grants Program](https://grants.osmosis.zone/). This new version adds a web dashboard, prometheus exporter, telegram and discord notifications, multi-chain support, more granular alerting, and more types of alerts.

![dashboard screenshot](docs/dash.png)

## Documentation

The [documentation](docs/README.md) is a work-in-progress.

## Runtime options:

```
$ tenderduty -h
Usage of tenderduty:
  -example-config
    	print the an example config.yml and exit
  -f string
    	configuration file to use (default "config.yml")
  -state string
    	file for storing state between restarts (default ".tenderduty-state.json")
  -cc string
    	directory containing additional chain specific configurations (default "chains.d")
```

## Installing

Detailed installation info is in the [installation doc.](docs/install.md)

30 second quickstart if you already have Docker installed:

```
mkdir tenderduty && cd tenderduty
docker run --rm ghcr.io/blockpane/tenderduty:latest -example-config >config.yml
# edit config.yml and add chains, notification methods etc.
docker run -d --name tenderduty -p "8888:8888" -p "28686:28686" --restart unless-stopped -v $(pwd)/config.yml:/var/lib/tenderduty/config.yml ghcr.io/blockpane/tenderduty:latest
docker logs -f --tail 20 tenderduty
```


## Split Configuration

For validators with many chains, chain specific configuration may be split into additional files and placed into the directory "chains.d".

This directory can be changed with the -cc option

The user friendly chain label will be taken from the name of the file.  

For example:

```
chains.d/Juno.yml -> Juno
chains.d/Lum Network.yml -> Lum Network
```

Configuration inside chains.d/Network.yml will be the YAML contents without the chain label.

For example start directly with:

```
chain_id: demo-1
    valoper_address: demovaloper...
```

## Contributions

Contributions are welcome, please open pull requests against the 'develop' branch, not main.

