package conf

func GetTronGrpcNode() string {
	if cfg.TronGrpcNode != "" {

		return cfg.TronGrpcNode
	}

	return defaultTronGrpcNode
}

func GetAptosRpcNode() string {
	if cfg.AptosRpcNode != "" {
		return cfg.AptosRpcNode
	}

	return defaultAptosRpcEndpoint
}

func GetSolanaRpcEndpoint() string {
	if cfg.EvmRpc.Solana != "" {

		return cfg.EvmRpc.Solana
	}

	return defaultSolanaRpcEndpoint
}

func GetXlayerRpcEndpoint() string {
	if cfg.EvmRpc.Xlayer != "" {

		return cfg.EvmRpc.Xlayer
	}

	return defaultXlayerRpcEndpoint
}

func GetBscRpcEndpoint() string {
	if cfg.EvmRpc.Bsc != "" {

		return cfg.EvmRpc.Bsc
	}

	return defaultBscRpcEndpoint
}

func GetPolygonRpcEndpoint() string {
	if cfg.EvmRpc.Polygon != "" {

		return cfg.EvmRpc.Polygon
	}

	return defaultPolygonRpcEndpoint
}

func GetArbitrumRpcEndpoint() string {
	if cfg.EvmRpc.Arbitrum != "" {

		return cfg.EvmRpc.Arbitrum
	}

	return defaultArbitrumRpcEndpoint
}

func GetEthereumRpcEndpoint() string {
	if cfg.EvmRpc.Ethereum != "" {

		return cfg.EvmRpc.Ethereum
	}

	return defaultEthereumRpcEndpoint
}

func GetBaseRpcEndpoint() string {
	if cfg.EvmRpc.Base != "" {

		return cfg.EvmRpc.Base
	}

	return defaultBaseRpcEndpoint
}
