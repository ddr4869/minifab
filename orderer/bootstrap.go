package orderer

// // CreateGenesisConfigFromConfigTx configtx.yaml 파일에서 ConfigTx 생성
// func CreateGenesisConfigFromConfigTx(configTxPath string, profile string) (*configtx.SystemChannelInfo, error) {
// 	if configTxPath == "" {
// 		return nil, errors.Errorf("configtx path cannot be empty")
// 	}

// 	// configtx.yaml 파일 존재 확인
// 	if _, err := os.Stat(configTxPath); os.IsNotExist(err) {
// 		return nil, errors.Errorf("configtx file does not exist: %s", configTxPath)
// 	}

// 	// configtx.yaml 파일 읽기
// 	data, err := os.ReadFile(configTxPath)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to read configtx file")
// 	}

// 	// YAML 파싱
// 	var configTx configtx.ConfigTx
// 	if err := yaml.Unmarshal(data, &configTx); err != nil {
// 		return nil, errors.Wrap(err, "failed to parse configtx YAML")
// 	}

// 	// ConfigTxYAML을 ConfigTx로 변환
// 	genesisConfig, err := configTx.GetSystemChannelInfo(profile)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
// 	}

// 	logger.Infof("Successfully loaded configuration from %s with profile %s", configTxPath, profile)

// 	return genesisConfig, nil
// }
