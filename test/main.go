package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	pb_common "github.com/ddr4869/minifab/proto/common"
	"google.golang.org/protobuf/proto"
)

// SystemChannelConfig 시스템 채널 설정 정보
type SystemChannelConfig struct {
	Orderer     OrdererConfig      `json:"Orderer"`
	Consortiums []ConsortiumConfig `json:"Consortiums"`
}

// OrdererConfig Orderer 설정 정보
type OrdererConfig struct {
	BatchTimeout string       `json:"BatchTimeout"`
	BatchSize    BatchSize    `json:"BatchSize"`
	Organization Organization `json:"Organization"`
}

// BatchSize 배치 크기 설정
type BatchSize struct {
	MaxMessageCount   int    `json:"MaxMessageCount"`
	AbsoluteMaxBytes  string `json:"AbsoluteMaxBytes"`
	PreferredMaxBytes string `json:"PreferredMaxBytes"`
}

// Organization 조직 정보
type Organization struct {
	Name             string       `json:"Name"`
	ID               string       `json:"ID"`
	MSPDir           string       `json:"MSPDir"`
	OrdererEndpoints []string     `json:"OrdererEndpoints"`
	AnchorPeers      []AnchorPeer `json:"AnchorPeers,omitempty"`
}

// AnchorPeer 앵커 피어 정보
type AnchorPeer struct {
	Host string `json:"Host"`
	Port int    `json:"Port"`
}

// ConsortiumConfig 컨소시엄 설정 정보
type ConsortiumConfig struct {
	Name             string       `json:"Name"`
	ID               string       `json:"ID"`
	MSPDir           string       `json:"MSPDir"`
	OrdererEndpoints []string     `json:"OrdererEndpoints"`
	AnchorPeers      []AnchorPeer `json:"AnchorPeers"`
}

func main() {
	fmt.Println(" Genesis Block 파일 읽기 시작...")

	// 1. genesis.block 파일 읽기
	genesisBlock, err := loadGenesisBlock("./blocks/genesis.block")
	if err != nil {
		log.Fatalf("❌ Genesis block 파일 읽기 실패: %v", err)
	}

	fmt.Printf("✅ Genesis Block 로드 성공\n")
	fmt.Printf("   - 채널 ID: %s\n", genesisBlock.ChannelId)
	fmt.Printf("   - 저장 시간: %s\n", genesisBlock.StoredAt)
	fmt.Printf("   - 커밋 상태: %t\n", genesisBlock.IsCommitted)
	fmt.Printf("   - 블록 해시: %s\n", genesisBlock.BlockHash)

	// 2. 블록 데이터에서 시스템 채널 설정 추출
	systemConfig, err := extractSystemChannelConfig(genesisBlock.Block)
	if err != nil {
		log.Fatalf("❌ 시스템 채널 설정 추출 실패: %v", err)
	}

	// 3. 시스템 채널 정보 출력
	printSystemChannelInfo(systemConfig)
}

// loadGenesisBlock genesis.block 파일을 읽어서 GenesisBlock 객체로 반환
func loadGenesisBlock(filePath string) (*pb_common.ConfigBlock, error) {
	// 파일 읽기
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	// protobuf 역직렬화
	genesisBlock := &pb_common.ConfigBlock{}
	if err := proto.Unmarshal(data, genesisBlock); err != nil {
		return nil, fmt.Errorf("protobuf 역직렬화 실패: %w", err)
	}

	return genesisBlock, nil
}

// extractSystemChannelConfig 블록에서 시스템 채널 설정 정보 추출
func extractSystemChannelConfig(block *pb_common.Block) (*SystemChannelConfig, error) {
	if block == nil || block.Data == nil || len(block.Data.Transactions) == 0 {
		return nil, fmt.Errorf("블록 데이터가 없거나 트랜잭션이 없습니다")
	}

	// 첫 번째 트랜잭션에서 설정 데이터 추출
	configData := block.Data.Transactions[0]

	fmt.Printf("🔍 트랜잭션 데이터 길이: %d bytes\n", len(configData))
	fmt.Printf("🔍 트랜잭션 데이터 (처음 100바이트): %s\n", string(configData[:min(100, len(configData))]))

	// 데이터가 이미 JSON인지 확인
	var systemConfig SystemChannelConfig
	if err := json.Unmarshal(configData, &systemConfig); err == nil {
		fmt.Println("✅ JSON 파싱 성공 (직접 파싱)")
		return &systemConfig, nil
	}

	// Base64 디코딩 시도
	decodedData, err := base64.StdEncoding.DecodeString(string(configData))
	if err != nil {
		// Base64 디코딩 실패 시 원본 데이터를 문자열로 출력
		fmt.Printf("⚠️  Base64 디코딩 실패, 원본 데이터: %s\n", string(configData))
		return nil, fmt.Errorf("Base64 디코딩 실패: %w", err)
	}

	fmt.Printf("✅ Base64 디코딩 성공, 디코딩된 데이터 길이: %d bytes\n", len(decodedData))

	// 디코딩된 데이터를 JSON으로 파싱
	if err := json.Unmarshal(decodedData, &systemConfig); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %w", err)
	}

	return &systemConfig, nil
}

// printSystemChannelInfo 시스템 채널 정보 출력
func printSystemChannelInfo(config *SystemChannelConfig) {
	fmt.Println("\n📋 시스템 채널 설정 정보:")

	// Orderer 정보
	fmt.Printf("🔧 Orderer 설정:\n")
	fmt.Printf("   - 배치 타임아웃: %s\n", config.Orderer.BatchTimeout)
	fmt.Printf("   - 최대 메시지 수: %d\n", config.Orderer.BatchSize.MaxMessageCount)
	fmt.Printf("   - 최대 바이트: %s\n", config.Orderer.BatchSize.AbsoluteMaxBytes)
	fmt.Printf("   - 선호 바이트: %s\n", config.Orderer.BatchSize.PreferredMaxBytes)

	// Orderer 조직 정보
	fmt.Printf("   - 조직명: %s\n", config.Orderer.Organization.Name)
	fmt.Printf("   - MSP ID: %s\n", config.Orderer.Organization.ID)
	fmt.Printf("   - MSP 디렉토리: %s\n", config.Orderer.Organization.MSPDir)
	fmt.Printf("   - Orderer 엔드포인트: %v\n", config.Orderer.Organization.OrdererEndpoints)

	// 컨소시엄 정보
	fmt.Printf("\n🏢 컨소시엄 정보:\n")
	for i, consortium := range config.Consortiums {
		fmt.Printf("   [%d] %s (ID: %s)\n", i+1, consortium.Name, consortium.ID)
		fmt.Printf("       - MSP 디렉토리: %s\n", consortium.MSPDir)
		fmt.Printf("       - Orderer 엔드포인트: %v\n", consortium.OrdererEndpoints)

		if len(consortium.AnchorPeers) > 0 {
			fmt.Printf("       - 앵커 피어:\n")
			for _, peer := range consortium.AnchorPeers {
				fmt.Printf("         * %s:%d\n", peer.Host, peer.Port)
			}
		}
	}

}

// saveSystemConfigToJSON 시스템 채널 설정을 JSON 파일로 저장
func saveSystemConfigToJSON(config *SystemChannelConfig, filePath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 마샬링 실패: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}

	return nil
}

// min 함수 (Go 1.21 이전 버전용)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
