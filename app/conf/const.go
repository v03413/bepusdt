package conf

const (
	defaultExpireTime       = 600      // 订单默认有效期 10分钟
	DefaultUsdtCnyRate      = 6.4      // 默认USDT基准汇率
	DefaultTrxCnyRate       = 0.95     // 默认TRX基准汇率
	defaultAuthToken        = "123234" // 默认授权码
	defaultListen           = ":8080"  // 默认监听地址
	defaultPaymentMinAmount = 0.01
	defaultPaymentMaxAmount = 99999
	defaultUsdtAtomicity    = 0.01 // 原子精度
	defaultTrxAtomicity     = 0.01

	defaultTronGrpcNode        = "18.141.79.38:50051"                      // 默认GRPC节点
	defaultBscRpcEndpoint      = "https://bsc-dataseed.bnbchain.org/"      // 默认BSC RPC节点
	defaultSolanaRpcEndpoint   = "https://solana-rpc.publicnode.com/"      // 默认Solana RPC节点 官方是 https://api.mainnet-beta.solana.com/ 但存在速率限制
	defaultXlayerRpcEndpoint   = "https://xlayerrpc.okx.com/"              // 默认Xlayer RPC节点
	defaultPolygonRpcEndpoint  = "https://polygon-bor-rpc.publicnode.com/" // 默认Polygon RPC节点 官方 https://polygon-rpc.com 存在速率限制
	defaultArbitrumRpcEndpoint = "https://arb1.arbitrum.io/rpc"            // 默认Arbitrum One RPC节点
	defaultEthereumRpcEndpoint = "https://ethereum.publicnode.com/"        // 默认Ethereum RPC节点
	defaultAptosRpcEndpoint    = "https://aptos-rest.publicnode.com/"      // 默认Aptos RPC节点
	defaultOutputLog           = "/var/log/bepusdt.log"                    // 默认日志输出文件
	defaultSqlitePath          = "/var/lib/bepusdt/sqlite.db"              // 默认数据库文件
)

const (
	UsdtErc20    = "0xdac17f958d2ee523a2206206994597c13d831ec7"                         // Eth USDT合约地址
	UsdtBep20    = "0x55d398326f99059ff775485246999027b3197955"                         // BSC USDT合约地址
	UsdtXlayer   = "0x1e4a5963abfd975d8c9021ce480b42188849d41d"                         // Xlayer USDT合约地址
	UsdtPolygon  = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"                         // Polygon USDT合约地址
	UsdtArbitrum = "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"                         // Arbitum One USDT合约地址
	UsdtSolana   = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"                       // Solana USDT合约地址
	SolSplToken  = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"                        // Solana SPL Token合约地址
	UsdtAptos    = "0x357b0b74bc833e95a115ad22604854d6b0fca151cecd94111770e5d6ffc9dc2b" // Aptos USDT合约地址

	UsdcErc20    = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	UsdcPolygon  = "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359"
	UsdcXlayer   = "0x74b7f16337b8972027f6196a17a631ac6de26d22"
	UsdcArbitrum = "0xaf88d065e77c8cc2239327c5edb3a432268e5831"
	UsdcBep20    = "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"
	UsdcSolana   = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

const (
	UsdtTronDecimals     = -6  // USDT Tron小数位数
	UsdtBscDecimals      = -18 // USDT BEP20小数位数
	UsdtEthDecimals      = -6  // USDT ERC20小数位数
	UsdtXlayerDecimals   = -6  // USDT Xlayer小数位数
	UsdtPolygonDecimals  = -6  // USDT Polygon小数位数
	UsdtArbitrumDecimals = -6  // USDT Arbitrum小数位数
	UsdtAptosDecimals    = -6  // USDT Aptos小数位数

	UsdcEthDecimals      = -6
	UsdcPolygonDecimals  = -6
	UsdcXlayerDecimals   = -6
	UsdcArbitrumDecimals = -6
	UsdcBscDecimals      = -18
	UsdcTronDecimals     = -6
)

const (
	NotifyMaxRetry     = 10   // 最大重试次数，订单回调失败、Webhook失败
	BlockHeightMaxDiff = 1000 // 区块高度最大差值，超过此值则以当前区块高度为准，重新开始扫描
)
