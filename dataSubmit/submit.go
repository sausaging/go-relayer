package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ava-labs/hypersdk/rpc"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	config "github.com/sausaging/go-relayer/internal"

	lrpc "github.com/sausaging/hyper-pvzk/rpc"
)

type Block struct {
	Data []byte
}

// submitData creates a transaction and makes a Avail data submission
func submitData(data []byte, ApiURL string, Seed string, appID int, meta *types.Metadata, api *gsrpc.SubstrateAPI) error {
	c, err := types.NewCall(meta, "DataAvailability.submit_data", types.NewBytes(data))
	if err != nil {
		return fmt.Errorf("cannot create new call:%w", err)
	}

	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return fmt.Errorf("cannot get block hash:%w", err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return fmt.Errorf("cannot get runtime version:%w", err)
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		return fmt.Errorf("cannot create LeyPair:%w", err)
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		return fmt.Errorf("cannot create storage key:%w", err)
	}

	var accountInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return fmt.Errorf("cannot get latest storage:%w", err)
	}

	nonce := uint32(accountInfo.Nonce)
	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		AppID:              types.NewUCompactFromUInt(uint64(appID)),
		TransactionVersion: rv.TransactionVersion,
	}

	err = ext.Sign(keyringPair, o)
	if err != nil {
		return fmt.Errorf("cannot sign:%w", err)
	}

	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return fmt.Errorf("cannot submit extrinsic:%w", err)
	}
	fmt.Printf("Data submitted by Alice: %v against appID %v  sent with hash %#x\n", data, appID, hash)

	return nil
}

func main() {
	var configJSON string
	var client *rpc.WebSocketClient
	var config config.Config
	flag.StringVar(&configJSON, "config", "", "config json file")
	flag.Parse()
	if configJSON == "" {
		log.Println("No config file provided. Exiting...")
		os.Exit(0)
	}
	err := config.GetConfig(configJSON)

	if err != nil {
		panic(fmt.Sprintf("cannot get config:%v", err))
	}
	nodeUrl := config.NodeURL
	jsonClient := rpc.NewJSONRPCClient(nodeUrl)
	network, _, chainId, err := jsonClient.Network(context.TODO())
	if err != nil {
		log.Println("cannot get network: %v", err)
		os.Exit(0)
	}
	lcli := lrpc.NewJSONRPCClient(nodeUrl, network, chainId)
	parser, err := lcli.Parser(context.TODO())
	if err != nil {
		log.Println("cannot create parser")
		os.Exit(0)
	}

	// Create a new Hyperspace client
	client, err = rpc.NewWebSocketClient(nodeUrl, 10*time.Second, 10, 1024*1024)
	if err != nil {
		panic(fmt.Sprintf("cannot create client:%v", err))
	}

	err = client.RegisterBlocks()
	if err != nil {
		panic(fmt.Sprintf("cannot register for blocks: %v", err))
	}

	api, err := gsrpc.NewSubstrateAPI(config.ApiURL)
	if err != nil {
		panic(fmt.Sprintf("cannot create api:%v", err))
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		panic(fmt.Sprintf("cannot get metadata:%v", err))
	}

	appID := 0

	AppID := config.AppID

	if AppID != 0 {
		appID = AppID
	}
	nonce := uint32(1532)
	var blockList []Block
	for {
		block, _, _, err := client.ListenBlock(context.TODO(), parser)
		if err != nil {
			log.Printf("Error listening to block: %v\n", err)
			break
		}
		blockData, err := block.Marshal()
		if err != nil {
			log.Printf("Error marshalling block: %v\n", err)
			continue
		}

		blockList = append(blockList, Block{Data: blockData})

		if len(blockList) == 4 {
			aggregateBlockData, err := aggregateBlockData(blockList)
			if err != nil {
				log.Printf("Error aggregating block data: %v\n", err)
				continue
			}
			err = submitData(aggregateBlockData, config.ApiURL, config.Seed, appID, meta, api)
			if err != nil {
				log.Printf("Error submitting proof: %v\n", err)
			} else {
				log.Println("Successfully submitted proof for block:", block.Hght)
				nonce++
			}
			blockList = []Block{}
		}
	}
	err = client.Close()
	if err != nil {
		log.Printf("Error closing WebSocket client: %v\n", err)
	}
}

func aggregateBlockData(data []Block) ([]byte, error) {
	var allBytes []byte
	for _, block := range data {
	  allBytes = append(allBytes, block.Data...)
	  // combine with a series of 0's to separate the blocks
	  allBytes = append(allBytes, []byte{0, 0, 0, 0, 0 , 0}...)
	}
	return allBytes, nil
  }
