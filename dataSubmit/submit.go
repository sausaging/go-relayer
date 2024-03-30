package submit

import (
	// "avail-gsrpc-examples/internal/config"
	// "avail-gsrpc-examples/internal/extrinsics"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
	"config"

	"github.com/ava-labs/hypersdk/rpc"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	lrpc "github.com/sausaging/hyper-pvzk/rpc"
)

// submitData creates a transaction and makes a Avail data submission
func submitData(data []byte, ApiURL string, Seed string, appID int, meta *types.Metadata, api *gsrpc.SubstrateAPI, nonce uint32) error {
	// api, err := gsrpc.NewSubstrateAPI(ApiURL)
	// if err != nil {
	// 	return fmt.Errorf("cannot create api:%w", err)
	// }

	// meta, err := api.RPC.State.GetMetadataLatest()
	// if err != nil {
	// 	return fmt.Errorf("cannot get metadata:%w", err)
	// }

	// // Set data and appID according to need
	// appID := 0

	// // if app id is greater than 0 then it must be created before submitting data
	// if AppID != 0 {
	// 	appID = AppID
	// }

	c, err := types.NewCall(meta, "DataAvailability.submit_data", types.NewBytes(data))
	if err != nil {
		return fmt.Errorf("cannot create new call:%w", err)
	}

	// Create the extrinsic
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

	// nonce := uint32(accountInfo.Nonce)
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

	// Sign the transaction using Alice's default account
	err = ext.Sign(keyringPair, o)
	if err != nil {
		return fmt.Errorf("cannot sign:%w", err)
	}

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return fmt.Errorf("cannot submit extrinsic:%w", err)
	}
	fmt.Printf("Data submitted by Alice: %v against appID %v  sent with hash %#x\n", data, appID, hash)

	return nil
}

func main() {
	var configJSON string
	node_url :=
		"http://127.0.0.1:36477/ext/bc/k57dGnG6YUyuNiCLEBcgvaoq9NC9phzPfZmH1xGfyQ9ftsKo8"
	var client *rpc.WebSocketClient
	var config config.Config
	flag.StringVar(&configJSON, "config", "", "config json file")
	flag.Parse()
	jsonClient := rpc.NewJSONRPCClient(node_url)
	network, _, chainId, err := jsonClient.Network(context.TODO())
	if err != nil {
		log.Println("cannot get network: %v", err)
		os.Exit(0)
	}
	lcli := lrpc.NewJSONRPCClient(node_url, network, chainId)
	parser, err := lcli.Parser(context.TODO())
	if err != nil {
		log.Println("cannot create parser")
		os.Exit(0)
	}
	if configJSON == "" {
		log.Println("No config file provided. Exiting...")
		os.Exit(0)
	}

	err = config.GetConfig(configJSON)
	if err != nil {
		panic(fmt.Sprintf("cannot get config:%v", err))
	}

	// Create a new Hyperspace client
	client, err = rpc.NewWebSocketClient(node_url, 10*time.Second, 10, 1024*1024)
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

	// Set data and appID according to need
	appID := 0

	AppID := config.AppID
	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}
	nonce := uint32(1532)
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

		// Call your submitProof function with the block data
		err = submitData(blockData, config.ApiURL, config.Seed, appID, meta, api, nonce)
		if err != nil {
			log.Printf("Error submitting proof: %v\n", err)
		} else {
			log.Println("Successfully submitted proof for block:", block.Hght)
			nonce++
		}
	}
	err = client.Close()
	if err != nil {
		log.Printf("Error closing WebSocket client: %v\n", err)
	}
}
