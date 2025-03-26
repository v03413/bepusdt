package conf

type Conf struct {
	AppUri             string `toml:"app_uri"`
	AuthToken          string `toml:"auth_token"`
	Listen             string `toml:"listen"`
	OutputLog          string `toml:"output_log"`
	StaticPath         string `toml:"static_path"`
	SqlitePath         string `toml:"sqlite_path"`
	TronGrpcNode       string `toml:"tron_grpc_node"`
	PolygonRpcEndpoint string `toml:"polygon_rpc_endpoint"`
	Pay                struct {
		TrxAtom          float64  `toml:"trx_atom"`
		TrxRate          string   `toml:"trx_rate"`
		UsdtAtom         float64  `toml:"usdt_atom"`
		UsdtRate         string   `toml:"usdt_rate"`
		ExpireTime       int      `toml:"expire_time"`
		WalletAddress    []string `toml:"wallet_address"`
		TradeIsConfirmed bool     `toml:"trade_is_confirmed"`
		PaymentAmountMin float64  `toml:"payment_amount_min"`
		PaymentAmountMax float64  `toml:"payment_amount_max"`
	} `toml:"pay"`
	Bot struct {
		Token   string `toml:"token"`
		AdminID int64  `toml:"admin_id"`
		GroupID string `toml:"group_id"`
	} `toml:"bot"`
}
