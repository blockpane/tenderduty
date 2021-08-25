package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"log"
	"os"
	"strings"
	"time"
)

var (
	alertThreshold, alertReminder int
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var endpoints, consAddr, pagerDuty string
	var testPD bool
	flag.StringVar(&endpoints, "u", "", "Required: comma seperated list of tendermint RPC urls (http:// or unix://)")
	flag.StringVar(&consAddr, "c", "", "Required: consensus address (valcons) to monitor '<gaiad> tendermint show-address'")
	flag.StringVar(&pagerDuty, "p", "", "Required: pagerduty api key")
	flag.IntVar(&alertThreshold, "threshold", 3, "alert threshold for missed precommits")
	flag.IntVar(&alertReminder, "reminder", 1200, "send additional alert every <reminder> blocks if still missing")
	flag.BoolVar(&testPD, "test", false, "send a test alert to pager duty, wait 10 seconds, resolve the incident and exit")
	flag.Parse()

	rpcs := strings.Split(strings.ReplaceAll(endpoints, " ", ""), ",")
	switch "" {
	case rpcs[0]:
		flag.PrintDefaults()
		log.Fatal("No endpoints provided!")
	case consAddr:
		flag.PrintDefaults()
		log.Fatal("No valconspub provided!")
	case pagerDuty:
		flag.PrintDefaults()
		log.Fatal("No pagerduty key provided!")
	}

	if !strings.Contains(consAddr, "valcons") {
		flag.PrintDefaults()
		log.Fatal("expected 'valcons' in the consensus key")
	}

	if testPD {
		log.Println("Sending trigger event")
		err := notifyPagerduty(false, "ALERT cosmopager test event", consAddr, pagerDuty)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(10*time.Second)
		log.Println("Sending resolve event")
		err = notifyPagerduty(true, "RESOLVED cosmopager test event", consAddr, pagerDuty)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	notifications := make(chan string)
	go func() {
		for {
			func(){
				client, err := connect(rpcs)
				if err != nil {
					return
				}
				watchCommits(client, consAddr, notifications)
			}()
			time.Sleep(3*time.Second)
			log.Println("attempting to reconnect")
		}
	}()

	for n := range notifications {
		log.Println(n)
		if err := notifyPagerduty(strings.HasPrefix(n, "RESOLVED"), n, consAddr, pagerDuty); err != nil {
			log.Println(err)
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
	_, err = pagerduty.ManageEvent(pagerduty.V2Event{
		RoutingKey: key,
		Action:     action,
		DedupKey:   producer,
		Payload:    &pagerduty.V2Payload{
			Summary:   message,
			Source:    producer,
			Severity:  sev,
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
		log.Println("could not start ws client", err)
		return nil, err
	}
	log.Println("connecting to", endpoint)
	return client, err
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
		log.Println("could not get blockchain status", err)
		return
	}
	network := status.NodeInfo.Network
	log.Println("connected to", network)

	var isActive bool

	var myValidator *types.Validator
	page, perPage := 1, 500
	valSet, err := client.Validators(ctx, &status.SyncInfo.LatestBlockHeight, &page, &perPage)
	if err != nil {
		log.Println("could not get current validator set", err)
		return
	}

	_, consPubBytes, _ := bech32.DecodeAndConvert(consAddr)
	for i := range valSet.Validators {
		if bytes.Equal(valSet.Validators[i].Address.Bytes(), consPubBytes) {
			myValidator = valSet.Validators[i]
			isActive = true
			log.Printf("found %s in the active validator set.", consAddr)
			break
		}
	}

	if myValidator == nil {
		log.Printf("could not find %s in current validator set, disconnecting and will retry in 1 minute", consAddr)
		_ = client.Stop()
		time.Sleep(time.Minute)
		return
	}

	query := "tm.event = 'NewBlock'"
	blockEvent, err := client.Subscribe(ctx, "block-client", query)
	if err != nil {
		log.Println("could not subscribe to block events on ws", err)
		return
	}

	query = "tm.event = 'ValidatorSetUpdates'"
	valUpdates, e := client.Subscribe(ctx, "validator-client", query)
	if err != nil {
		log.Println("could not subscribe to validator events on ws", err)
		return
	}

	// watchdog ticker
	alive := time.NewTicker(4*time.Minute)

	var currentBlock, aliveBlock int64
	var missingCount int

	log.Println("watching for missed precommits")
	for {
		select {
		case <-client.Quit():
			log.Println("client quit")
			return

		case evt := <-blockEvent:
			if !isActive {
				continue
			}
			block, ok := evt.Data.(types.EventDataNewBlock)
			if !ok {
				log.Println("got the wrong event type")
				return
			}
			currentBlock = block.Block.Height
			missed := true
			for _, sig := range block.Block.LastCommit.Signatures {
				if sig.ValidatorAddress.String() == myValidator.Address.String() {
					if missingCount >= alertThreshold {
						notifications <- "RESOLVED validator is signing blocks on "+network
					}
					missingCount = 0
					missed = false
					if currentBlock % 30 == 0 {
						log.Println("block", currentBlock)
					}
					break
				}
			}
			if missed {
				missingCount += 1
				if missingCount == alertThreshold || missingCount % alertReminder == 0 {
					notifications <- fmt.Sprintf("ALERT validator has missed %d blocks on %s", missingCount, network)
				}
				log.Println("missed a precommit at height:", currentBlock)
			}

		case evt := <-valUpdates:
			update, ok := evt.Data.(types.EventDataValidatorSetUpdates)
			if !ok {
				log.Println("got the wrong event type for a validator update")
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
				notifications <- "ALERT validator is not in the active set on "+network
			}
			if isActive && !wasActive {
				notifications <- "RESOLVED validator is now in the active set on "+network
			}

		case <-alive.C:
			if currentBlock <= aliveBlock {
				log.Println("have not seen a new block in 4 minutes, reconnecting")
				return
			}
			aliveBlock = currentBlock
			cx, cn := context.WithTimeout(context.Background(), 2*time.Second)
			status, e = client.Status(cx)
			cn()
			if e != nil {
				log.Println("could not check sync status", e)
				return
			}
			if status.SyncInfo.CatchingUp {
				log.Println("node is syncing")
				return
			}
		}
	}
}

