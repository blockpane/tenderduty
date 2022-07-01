package tenderduty

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// altValopers is used to get a bech32 prefix for chains using non-standard naming
var altValopers = &valoperOverrides{
	Prefixes: map[string]string{
		"ival": "ica", // Iris hub

		// TODO: was told tgrade also has a custom prefix, but not sure what the pair is
		//"tval": "tvalcons",
	},
}

type valoperOverrides struct {
	sync.RWMutex
	Prefixes    map[string]string `json:"prefixes"`
	LastUpdated time.Time         `json:"last_updated"`
}

func (vo *valoperOverrides) getAltPrefix(oper string) (prefix string, ok bool) {
	split := strings.Split(oper, "1")
	if len(split) == 0 {
		return "", false
	}
	vo.RLock()
	defer vo.RUnlock()
	return altValopers.Prefixes[split[0]], altValopers.Prefixes[split[0]] != ""
}

// cosmosPaths holds a mapping useful for finding public nodes from the cosmos directory from eco_stake
// it will be refreshed periodically.
var cosmosPaths = map[string]string{
	"Antora":                     "idep",
	"Oraichain":                  "oraichain",
	"agoric-3":                   "agoric",
	"akashnet-2":                 "akash",
	"arkh":                       "arkh",
	"axelar-dojo-1":              "axelar",
	"bitcanna-1":                 "bitcanna",
	"bitsong-2b":                 "bitsong",
	"bostrom":                    "bostrom",
	"carbon-1":                   "carbon",
	"cerberus-chain-1":           "cerberus",
	"cheqd-mainnet-1":            "cheqd",
	"chihuahua-1":                "chihuahua",
	"colosseum-1":                "firmachain",
	"columbus-5":                 "terra",
	"comdex-1":                   "comdex",
	"core-1":                     "persistence",
	"cosmoshub-4":                "cosmoshub",
	"crescent-1":                 "crescent",
	"cronosmainnet_25-1":         "cronos",
	"crypto-org-chain-mainnet-1": "cryptoorgchain",
	"cudos-1":                    "cudos",
	"darchub":                    "konstellation",
	"desmos-mainnet":             "desmos",
	"dig-1":                      "dig",
	"echelon_3000-3":             "echelon",
	"emoney-3":                   "emoney",
	"evmos_9001-2":               "evmos",
	"fetchhub-4":                 "fetchhub",
	"galaxy-1":                   "galaxy",
	"genesis_29-2":               "genesisl1",
	"gravity-bridge-3":           "gravitybridge",
	"impacthub-3":                "impacthub",
	"injective-1":                "injective",
	"iov-mainnet-ibc":            "starname",
	"irishub-1":                  "irisnet",
	"juno-1":                     "juno",
	"kava_2222-10":               "kava",
	"kichain-2":                  "kichain",
	"laozi-mainnet":              "bandchain",
	"likecoin-mainnet-2":         "likecoin",
	"logos_7002-1":               "logos",
	"lum-network-1":              "lumnetwork",
	"mainnet-3":                  "decentr",
	"mantle-1":                   "assetmantle",
	"meme-1":                     "meme",
	"microtick-1":                "microtick",
	"morocco-1":                  "chronicnetwork",
	"mythos_7001-1":              "mythos",
	"nomic-stakenet-2":           "nomic",
	"octa":                       "octa",
	"odin-mainnet-freya":         "odin",
	"osmosis-1":                  "osmosis",
	"panacea-3":                  "panacea",
	"phoenix-1":                  "terra2",
	"pio-mainnet-1":              "provenance",
	"regen-1":                    "regen",
	"secret-4":                   "secretnetwork",
	"sentinelhub-2":              "sentinel",
	"shentu-2.2":                 "shentu",
	"sifchain-1":                 "sifchain",
	"sommelier-3":                "sommelier",
	"stargaze-1":                 "stargaze",
	"thorchain-mainnet-v1":       "thorchain",
	"titan-1":                    "rizon",
	"umee-1":                     "umee",
	"vidulum-1":                  "vidulum",
}
var pathMux sync.Mutex

const registryJson = "https://chains.cosmos.directory/"
const publicRpcUrl = "https://rpc.cosmos.directory:443/"

// a trimmed down version only holding the info we need to create a lookup map
type registryResults struct {
	Chains []struct {
		Path    string `json:"path"`
		ChainId string `json:"chain_id"`
	} `json:"chains"`
}

// refreshRegistry updates the path map for public RPC endpoints for @eco_stake's public RPC proxy
func refreshRegistry() error {
	res, err := http.Get(registryJson)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	chains := &registryResults{}
	err = json.Unmarshal(body, chains)
	if err != nil {
		return err
	}
	if len(chains.Chains) == 0 {
		return errors.New("response had no chains")
	}
	pathMux.Lock()
	defer pathMux.Unlock()
	if cosmosPaths == nil {
		cosmosPaths = make(map[string]string)
	}
	for _, c := range chains.Chains {
		cosmosPaths[c.ChainId] = c.Path
	}
	return nil
}

func getRegistryUrl(chainid string) (url string, ok bool) {
	pathMux.Lock()
	defer pathMux.Unlock()
	return publicRpcUrl + cosmosPaths[chainid], cosmosPaths[chainid] != ""
}
