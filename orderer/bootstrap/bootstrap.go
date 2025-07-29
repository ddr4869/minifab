package bootstrap

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ddr4869/minifab/common/blockutil"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var (
	address      string
	mspID        string
	mspPath      string
	genesisFile  string
	configTxPath string
	profile      string
	bootstrap    bool
)

func Cmd() *cobra.Command {

	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap the blockchain network with genesis block",
		Long: `Bootstrap the blockchain network by generating and initializing the genesis block.
This command should be run once when setting up a new network.`,
		Run: runBootstrap,
	}

	bootstrapCmd.PersistentFlags().StringVar(&address, "address", "0.0.0.0:7050", "Orderer server address")
	bootstrapCmd.PersistentFlags().StringVar(&mspID, "mspid", "OrdererMSP", "MSP ID for orderer")
	bootstrapCmd.PersistentFlags().StringVar(&mspPath, "mspdir", "./ca/ca-client/orderer0/msp", "Path to MSP directory with certificates")

	// Bootstrap command flags
	bootstrapCmd.Flags().StringVar(&genesisFile, "genesisFile", "./config/genesis.json", "Path to save/load genesis block file")
	bootstrapCmd.Flags().StringVar(&configTxPath, "configtx", "./config/configtx.yaml", "Path to configtx.yaml file")
	bootstrapCmd.Flags().StringVar(&profile, "profile", "SystemChannel", "Profile name to use for genesis block")
	bootstrapCmd.Flags().BoolVar(&bootstrap, "bootstrap", false, "Bootstrap network with genesis block")

	return bootstrapCmd
}

func runBootstrap(cmd *cobra.Command, args []string) {
	logger.Info("Starting network bootstrap process...")

	// configtx.yaml에서 제네시스 설정 생성 (profile 인자 추가)
	genesisConfig, err := CreateGenesisConfigFromConfigTx(configTxPath, profile)
	if err != nil {
		logger.Fatalf("Failed to load configtx.yaml: %v", err)
	}

	logger.Info("Successfully loaded configuration from configtx.yaml")

	// 네트워크 부트스트랩 실행
	if err := bootstrapNetwork(genesisConfig); err != nil {
		logger.Fatalf("Failed to bootstrap network: %v", err)
	}

	logger.Info("Network bootstrap completed successfully!")
	logger.Infof("Configuration loaded from: %s", configTxPath)
	logger.Info("You can now start the orderer with: ./bin/orderer")
}

// CreateGenesisConfigFromConfigTx configtx.yaml 파일에서 ConfigTx 생성
func CreateGenesisConfigFromConfigTx(configTxPath string, profile string) (*configtx.SystemChannelInfo, error) {

	configTx, err := configtx.ConvertConfigtx(configTxPath, profile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx")
	}
	// ConfigTxYAML을 ConfigTx로 변환
	genesisConfig, err := configTx.GetSystemChannelInfo(profile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s with profile %s", configTxPath, profile)

	return genesisConfig, nil
}

func bootstrapNetwork(genesisConfig *configtx.SystemChannelInfo) error {

	err := generateGenesisBlock(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate genesis block")
	}

	logger.Info("Genesis block created and saved successfully")

	return nil
}

func generateGenesisBlock(genesisConfig *configtx.SystemChannelInfo) error {
	// 설정 트랜잭션 데이터 직렬화
	configTxData, err := json.Marshal(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis config")
	}

	header := &pb_common.BlockHeader{
		Number:       0,
		PreviousHash: nil,
		HeaderType:   pb_common.BlockType_BLOCK_TYPE_CONFIG,
	}

	blockData := &pb_common.BlockData{
		Transactions: [][]byte{
			configTxData,
		},
	}

	msp, err := msp.LoadMSPFromFiles(mspID, mspPath)
	if err != nil {
		return errors.Wrap(err, "failed to load MSP")
	}

	signer := msp.GetSigningIdentity()
	signature, err := signer.Sign(rand.Reader, configTxData, nil)
	if err != nil {
		return errors.Wrap(err, "failed to sign config tx")
	}

	metadata := &pb_common.BlockMetadata{
		CreatorCertificate: signer.GetCertificate().Raw,
		CreatorSignature:   signature,
		ValidationBitmap:   []byte{1},
		AccumulatedHash:    []byte{},
	}

	block := &pb_common.Block{
		Header:   header,
		Data:     blockData,
		Metadata: metadata,
	}

	header.CurrentBlockHash = blockutil.CalculateBlockHash(block)

	genesisBlock := &pb_common.GenesisBlock{
		Block:       block,
		ChannelId:   "SYSTEM_CHANNEL",
		StoredAt:    time.Now().Format(time.RFC3339),
		IsCommitted: true,
		BlockHash:   fmt.Sprintf("%x", header.CurrentBlockHash),
	}

	// protobuf로 직렬화
	protoData, err := proto.Marshal(genesisBlock)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block")
	}

	if err := os.WriteFile("./blocks/genesis.block", protoData, 0644); err != nil {
		return errors.Wrap(err, "failed to write genesis block file")
	}
	logger.Info("Genesis block created and saved at blocks/genesis.block successfully")

	jsonData, err := json.MarshalIndent(genesisBlock, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block to JSON")
	}

	if err := os.WriteFile("genesis.json", jsonData, 0644); err != nil {
		return errors.Wrap(err, "failed to write genesis JSON file")
	}
	logger.Info("Genesis info created and saved at genesis.json successfully")

	return nil
}
