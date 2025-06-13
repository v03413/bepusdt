package conf

func GetTronGrpcNode() string {
	if cfg.TronGrpcNode != "" {

		return cfg.TronGrpcNode
	}

	return defaultTronGrpcNode
}

func GetXlayerRpcEndpoint() string {
	if cfg.XlayerRpcEndpoint != "" {

		return cfg.XlayerRpcEndpoint
	}

	return defaultXlayerRpcEndpoint
}

func GetBscRpcEndpoint() string {
	if cfg.BscRpcEndpoint != "" {

		return cfg.BscRpcEndpoint
	}

	return defaultBscRpcEndpoint
}

func GetPolygonRpcEndpoint() string {
	if cfg.PolygonRpcEndpoint != "" {

		return cfg.PolygonRpcEndpoint
	}

	return defaultPolygonRpcEndpoint
}

func GetEthereumRpcEndpoint() string {
	if cfg.EthereumRpcEndpoint != "" {

		return cfg.EthereumRpcEndpoint
	}

	return defaultEthereumRpcEndpoint
}
