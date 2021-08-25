# TenderDuty

A PagerDuty notifier for Cosmos/Tendermint validators.

Features:

* Will send an alert if a certain threshold of missed pre-commits are seen.
* Alerts if the validator leaves the active set.
* Will resolve the alert once the validator is signing again.
* Accepts a list of RPC endpoints and randomly connects to one.

Install:

```shell
$ git clone https://github.com/blockpane/tenderduty.git
$ cd tenderduty
$ go build -ldflags "-s -w" -o tenderduty main.go
```

Options:

```
Usage of tenderduty:
  -c string
        Required: consensus address (valcons) to monitor '<gaiad> tendermint show-address'
  -p string
        Required: pagerduty api key
  -reminder int
        send additional alert every <reminder> blocks if still missing (default 1200)
  -test
        send a test alert to pager duty, wait 10 seconds, resolve the incident and exit
  -threshold int
        alert threshold for missed precommits (default 3)
  -u string
        Required: comma seperated list of tendermint RPC urls (http:// or unix://)
```

Example use:

```shell
tenderduty -c pagervalcons1vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv -u http://1.2.3.4:26657,http://3.4.5.6:26657 -p ffffffffffffffffffffffffffffffff
2021/08/25 13:48:02 main.go:126: connecting to http://1.2.3.4:26657
2021/08/25 13:48:02 main.go:149: connected to somechain-1
2021/08/25 13:48:02 main.go:166: found pagervalcons1vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv in the active validator set.
2021/08/25 13:48:02 main.go:198: watching for missed precommits
2021/08/25 13:48:29 main.go:224: block 917700
2021/08/25 13:51:45 main.go:224: block 917730
```