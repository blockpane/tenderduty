# TenderDuty

A [PagerDuty](https://github.com/PagerDuty/go-pagerduty) notifier for [Cosmos](https://github.com/cosmos/cosmos-sdk) / [Tendermint](https://github.com/tendermint/tendermint) validators.

This will probably only work on Tendermint 0.34.x chains.

Features:

* Will send an alert if a certain threshold of (consecutive) missed pre-commits are seen.
* Alerts if the validator leaves the active set.
* Will resolve the alert once the validator is signing again.
* Accepts a list of Tendermint RPC endpoints and randomly connects to one (_does not need to run on the validator node._)

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
$ tenderduty -c pagevalcons1... -u http://1.2.3.4:26657,http://3.4.5.6:26657 -p efghi...

2021/08/25 14:41:10 main.go:126: connecting to http://1.2.3.4:26657
2021/08/25 14:41:10 main.go:149: connected to pager-1
pager-1      | 2021/08/25 14:41:10 main.go:167: found pagevalcons1... in the active validator set.
pager-1      | 2021/08/25 14:41:10 main.go:199: watching for missed precommits
pager-1      | 2021/08/25 14:43:13 main.go:225: block 918210
...
pager-1      | 2021/08/25 16:35:37 main.go:235: missed a precommit at height: 919268
pager-1      | 2021/08/25 16:35:45 main.go:235: missed a precommit at height: 919269
2021/08/25 16:35:51 main.go:87: ALERT validator has missed 3 blocks on pager-1
pager-1      | 2021/08/25 16:35:51 main.go:235: missed a precommit at height: 919270
pager-1      | 2021/08/25 16:35:55 main.go:235: missed a precommit at height: 919271
pager-1      | 2021/08/25 16:36:01 main.go:235: missed a precommit at height: 919272
pager-1      | 2021/08/25 16:36:08 main.go:235: missed a precommit at height: 919273
pager-1      | 2021/08/25 16:36:14 main.go:235: missed a precommit at height: 919274
pager-1      | 2021/08/25 16:36:20 main.go:235: missed a precommit at height: 919275
2021/08/25 16:36:26 main.go:87: RESOLVED validator is signing blocks on pager-1
pager-1      | 2021/08/25 16:37:55 main.go:225: block 919290
pager-1      | 2021/08/25 16:41:05 main.go:225: block 919320
```

To find your consensus address (valcons):

```
gaiad tendermint show-address
```
