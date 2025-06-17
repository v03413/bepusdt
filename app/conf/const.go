package conf

const (
	defaultExpireTime          = 600      // 订单默认有效期 10分钟
	DefaultUsdtCnyRate         = 6.4      // 默认USDT基准汇率
	DefaultTrxCnyRate          = 0.95     // 默认TRX基准汇率
	defaultAuthToken           = "123234" // 默认授权码
	defaultListen              = ":8080"  // 默认监听地址
	defaultPaymentMinAmount    = 0.01
	defaultPaymentMaxAmount    = 99999
	defaultUsdtAtomicity       = 0.01 // 原子精度
	defaultTrxAtomicity        = 0.01
	defaultTronGrpcNode        = "18.141.79.38:50051"                 // 默认GRPC节点
	defaultBscRpcEndpoint      = "https://bsc-dataseed.bnbchain.org/" // 默认BSC RPC节点
	defaultXlayerRpcEndpoint   = "https://xlayerrpc.okx.com/"         // 默认Xlayer RPC节点
	defaultPolygonRpcEndpoint  = "https://polygon-rpc.com/"           // 默认Polygon RPC节点
	defaultEthereumRpcEndpoint = "https://ethereum.publicnode.com/"   // 默认Ethereum RPC节点
	defaultOutputLog           = "/var/log/bepusdt.log"               // 默认日志输出文件
	defaultSqlitePath          = "/var/lib/bepusdt/sqlite.db"         // 默认数据库文件
)

const (
	UsdtErc20   = "0xdac17f958d2ee523a2206206994597c13d831ec7" // Eth USDT合约地址
	UsdtBep20   = "0x55d398326f99059ff775485246999027b3197955" // BSC USDT合约地址
	UsdtXlayer  = "0x1e4a5963abfd975d8c9021ce480b42188849d41d" // Xlayer USDT合约地址
	UsdtPolygon = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f" // Polygon USDT合约地址
)

// UsdtDecimals USDT合约小数位数
const (
	UsdtBscDecimals     = -18 // USDT BEP20小数位数
	UsdtEthDecimals     = -6  // USDT ERC20小数位数
	UsdtXlayerDecimals  = -6  // USDT Xlayer小数位数
	UsdtPolygonDecimals = -6  // USDT Polygon小数位数
)
