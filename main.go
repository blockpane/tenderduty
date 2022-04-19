package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	alertThreshold, alertReminder int
	l                             = log.New(os.Stdout, fmt.Sprintf("%-12s | ", "tenderduty"), log.LstdFlags|log.Lshortfile)
	deadCounter, deadAfter        int
	network                       = "unknown"
)

func main() {
	rex := regexp.MustCompile(`[+_-]`)
	var endpoints, consAddr, pagerDuty, label string
	var testPD, alertNoEndpoints bool
	flag.StringVar(&endpoints, "u", "", "Required: comma seperated list of tendermint RPC urls (http:// or unix://)")
	flag.StringVar(&consAddr, "c", "", "Required: consensus address (valcons) to monitor '<gaiad> tendermint show-address'")
	flag.StringVar(&pagerDuty, "p", "", "Required: pagerduty v2 Events api key (32 characters, alphanumeric)")
	flag.StringVar(&label, "label", "", "Additional text to add to the alert title, added after chain ID string")
	flag.IntVar(&alertThreshold, "threshold", 3, "alert threshold for missed precommits")
	flag.IntVar(&alertReminder, "reminder", 1200, "send additional alert every <reminder> blocks if still missing")
	flag.IntVar(&deadAfter, "stalled", 10, "alert if minutes since last block exceeds this value")
	flag.BoolVar(&testPD, "test", false, "send a test alert to pager duty, wait 10 seconds, resolve the incident and exit")
	flag.BoolVar(&alertNoEndpoints, "alert-unavailable", false, "send a pagerduty alert if no RPC endpoints are available")
	flag.Parse()

	rpcs := strings.Split(strings.ReplaceAll(endpoints, " ", ""), ",")
	switch "" {
	case rpcs[0]:
		flag.PrintDefaults()
		l.Fatal("No endpoints provided!")
	case consAddr:
		flag.PrintDefaults()
		l.Fatal("No valconspub provided!")
	case pagerDuty:
		flag.PrintDefaults()
		l.Fatal("No pagerduty key provided!")
	}

	if rex.MatchString(pagerDuty) {
		fmt.Println("The Pagerduty key provided appears to be an Oauth token, not a V2 Events API key.")
		fmt.Println(`To find your pagerduty integration key
  * Within PagerDuty, go to Services --> Service Directory --> New Service
  * Give your service a name, select an escalation policy, and set an alert grouping preference
  * Select the PagerDuty Events API V2 Integration, hit Create Service
  * Copy the 32 character Integration Key on the tenderduty command line`)
		os.Exit(1)
	}

	if !strings.Contains(consAddr, "valcons") {
		l.Println("Warning: expected 'valcons' in the consensus key.")
	}

	if testPD {
		l.Println("Sending trigger event")
		err := notifyPagerduty(false, "ALERT tenderduty test event", consAddr, pagerDuty)
		if err != nil {
			l.Fatal(err)
		}
		time.Sleep(10 * time.Second)
		l.Println("Sending resolve event")
		err = notifyPagerduty(true, "RESOLVED tenderduty test event", consAddr, pagerDuty)
		if err != nil {
			l.Fatal(err)
		}
		os.Exit(0)
	}

	// ensure we have at least one working endpoint
	l.Printf("checking that at least 1 of %d endpoints are available", len(rpcs))
	count, e := checkEndpoints(rpcs, l)
	if e != nil {
		l.Fatal("FATAL:", e)
	}
	l.Printf("successfully connected to %d endpoints", count)
	notifications := make(chan string)
	connectionErrors := 0
	connectionAlarm := false
	go func() {
		for {
			func() {
				if !connectionAlarm && alertNoEndpoints && connectionErrors > len(rpcs) {
					l.Println("too many connection errors, checking for live RPC endpoints")
					_, e = checkEndpoints(rpcs, l)
					if e != nil {
						connectionAlarm = true
						notifications <- "ALERT tenderduty cannot connect to RPC on " + network
					}
				}
				client, err := connect(rpcs)
				if err != nil {
					connectionErrors += 1
					return
				}
				if connectionAlarm {
					connectionAlarm = false
					notifications <- "RESOLVED tenderduty connected to RPC on " + network
				}
				connectionErrors = 0
				watchCommits(client, consAddr, notifications)
			}()
			time.Sleep(3 * time.Second)
			l.Println("attempting to reconnect")
		}
	}()

	for n := range notifications {
		if label != "" {
			n = fmt.Sprintf("%s (%s)", n, label)
		}
		l.Println(n)
		if err := notifyPagerduty(strings.HasPrefix(n, "RESOLVED"), n, consAddr, pagerDuty); err != nil {
			l.Println(err)
		}
	}
}

func notifyPagerduty(resolved bool, message, producer, key string) (err error) {
	if key == "" {
		return nil
	}
	action := "trigger"
	sev := "error"
	if resolved {
		action = "resolve"
		sev = "info"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		RoutingKey: key,
		Action:     action,
		DedupKey:   producer,
		Payload: &pagerduty.V2Payload{
			Summary:  message,
			Source:   producer,
			Severity: sev,
		},
	})
	return
}

func connect(endpoints []string) (*rpchttp.HTTP, error) {
	// grab a random endpoint from our array:
	endpoint := endpoints[intn(len(endpoints))]
	client, _ := rpchttp.New(endpoint, "/websocket")
	err := client.Start()
	if err != nil {
		l.Println("could not start ws client", err)
		return nil, err
	}
	l.Println("connecting to", endpoint)
	return client, err
}

func checkEndpoints(endpoints []string, l *log.Logger) (available int, err error) {
	available = len(endpoints)
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	for i := range endpoints {
		endpoint := endpoints[i]
		client, _ := rpchttp.New(endpoint, "/websocket")
		e := client.Start()
		if e != nil {
			l.Println("Warning: could not connect to", endpoint, e)
			available -= 1
			continue
		}
		_, e = client.Status(ctx)
		if e != nil {
			l.Println("Warning: could get status of", endpoint, e)
			available -= 1
		}
	}
	if available == 0 {
		err = errors.New("no RPC endpoints are available")
	}
	return
}

func intn(mod int) int {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return int(binary.LittleEndian.Uint32(b)>>1) % mod
}

func watchCommits(client *rpchttp.HTTP, consAddr string, notifications chan string) {
	defer client.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var err error

	var status *coretypes.ResultStatus
	status, err = client.Status(ctx)
	if err != nil {
		l.Println("could not get blockchain status", err)
		return
	}
	network = status.NodeInfo.Network
	l.Println("connected to", network)

	// update logger once we know network name:
	l = log.New(os.Stdout, fmt.Sprintf("%-12s | ", network), log.LstdFlags|log.Lshortfile)

	var isActive bool

	var myValidator *types.Validator
	page, perPage := 1, 100 // have to use 100 due to abci bug where perPage is ignored when > 100
	valSet, err := client.Validators(ctx, &status.SyncInfo.LatestBlockHeight, &page, &perPage)
	if err != nil {
		l.Println("could not get current validator set", err)
		return
	}

	_, consPubBytes, err := bech32.DecodeAndConvert(consAddr)
	if err != nil {
		l.Fatal("valcons address is invalid:", err)
	}
	// use paging or we can't find validators ranked > 100
	repeat := 1
	if valSet.Total > 100 {
		repeat += valSet.Total / 100
	}
found:
	for j := 1; j <= repeat; j++ {
		valSet, err = client.Validators(ctx, &status.SyncInfo.LatestBlockHeight, &j, &perPage)
		if err != nil {
			l.Println("could not get current validator set", err)
			return
		}
		for i := range valSet.Validators {
			if bytes.Equal(valSet.Validators[i].Address.Bytes(), consPubBytes) {
				myValidator = valSet.Validators[i]
				isActive = true
				l.Printf("found %s in the active validator set.", consAddr)
				break found
			}
		}
	}

	if myValidator == nil {
		l.Printf("could not find %s in current validator set, disconnecting and will retry in 1 minute", consAddr)
		_ = client.Stop()
		time.Sleep(time.Minute)
		return
	}

	query := "tm.event = 'NewBlock'"
	blockEvent, err := client.Subscribe(ctx, "block-client", query)
	if err != nil {
		l.Println("could not subscribe to block events on ws", err)
		return
	}

	query = "tm.event = 'ValidatorSetUpdates'"
	valUpdates, err := client.Subscribe(ctx, "validator-client", query)
	if err != nil {
		l.Println("could not subscribe to validator events on ws", err)
		return
	}

	// watchdog ticker
	alive := time.NewTicker(time.Minute)

	var currentBlock, aliveBlock int64
	var missingCount int

	l.Println("watching for missed precommits")
	for {
		select {
		case <-client.Quit():
			l.Println("client quit")
			return

		case evt := <-blockEvent:
			if !isActive {
				continue
			}
			block, ok := evt.Data.(types.EventDataNewBlock)
			if !ok {
				l.Println("got the wrong event type")
				return
			}
			currentBlock = block.Block.Height
			missed := true
			for _, sig := range block.Block.LastCommit.Signatures {
				if sig.ValidatorAddress.String() == myValidator.Address.String() {
					if missingCount >= alertThreshold {
						notifications <- "RESOLVED validator is signing blocks on " + network
					}
					missingCount = 0
					missed = false
					if currentBlock%30 == 0 {
						l.Println("block", currentBlock)
					}
					break
				}
			}
			if missed {
				missingCount += 1
				if missingCount == alertThreshold || missingCount%alertReminder == 0 {
					notifications <- fmt.Sprintf("ALERT validator has missed %d blocks on %s", missingCount, network)
				}
				l.Println("missed a precommit at height:", currentBlock)
			}

		case evt := <-valUpdates:
			update, ok := evt.Data.(types.EventDataValidatorSetUpdates)
			if !ok {
				l.Println("got the wrong event type for a validator update")
				return
			}
			var wasActive = isActive
			for i := range update.ValidatorUpdates {
				if update.ValidatorUpdates[i].Address.String() == myValidator.String() {
					isActive = true
					break
				}
			}
			if !isActive && wasActive {
				notifications <- "ALERT validator is not in the active set on " + network
			}
			if isActive && !wasActive {
				notifications <- "RESOLVED validator is now in the active set on " + network
			}

		case <-alive.C:
			if currentBlock <= aliveBlock {
				if deadCounter == deadAfter {
					notifications <- fmt.Sprintf("ALERT have not seen a new block in %d minutes on %s", deadAfter, network)
				}
				l.Println("have not seen a new block in 1 minutes, reconnecting")
				deadCounter += 1
				return
			} else if deadCounter >= deadAfter {
				notifications <- "RESOLVED blocks are incrementing on " + network
			}
			deadCounter = 0
			aliveBlock = currentBlock
			cx, cn := context.WithTimeout(context.Background(), 2*time.Second)
			status, err = client.Status(cx)
			cn()
			if err != nil {
				l.Println("could not check sync status", err)
				return
			}
			if status.SyncInfo.CatchingUp {
				l.Println("node is syncing")
				return
			}
		}
	}
}
