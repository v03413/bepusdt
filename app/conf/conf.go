package conf

type Conf struct {
	AppUri       string `toml:"app_uri"`
	AuthToken    string `toml:"auth_token"`
	Listen       string `toml:"listen"`
	OutputLog    string `toml:"output_log"`
	StaticPath   string `toml:"static_path"`
	SqlitePath   string `toml:"sqlite_path"`
	TronGrpcNode string `toml:"tron_grpc_node"`
	AptosRpcNode string `toml:"aptos_rpc_node"`
	WebhookUrl   string `toml:"webhook_url"`
	Pay          struct {
		TrxAtom          float64  `toml:"trx_atom"`
		TrxRate          string   `toml:"trx_rate"`
		UsdtAtom         float64  `toml:"usdt_atom"`
		UsdcAtom         float64  `toml:"usdc_atom"`
		UsdtRate         string   `toml:"usdt_rate"`
		UsdcRate         string   `toml:"usdc_rate"`
		ExpireTime       int      `toml:"expire_time"`
		WalletAddress    []string `toml:"wallet_address"`
		TradeIsConfirmed bool     `toml:"trade_is_confirmed"`
		PaymentAmountMin float64  `toml:"payment_amount_min"`
		PaymentAmountMax float64  `toml:"payment_amount_max"`
	} `toml:"pay"`
	EvmRpc struct {
		Bsc      string `toml:"bsc"`
		Solana   string `toml:"solana"`
		Xlayer   string `toml:"xlayer"`
		Polygon  string `toml:"polygon"`
		Arbitrum string `toml:"arbitrum"`
		Ethereum string `toml:"ethereum"`
		Base     string `toml:"base"`
	} `toml:"evm_rpc"`
	Bot struct {
		Token   string `toml:"token"`
		AdminID int64  `toml:"admin_id"`
		GroupID string `toml:"group_id"`
	} `toml:"bot"`
}
