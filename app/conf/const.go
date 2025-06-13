package conf

const (
	defaultExpireTime          = 600      // 订单默认有效期 10分钟
	DefaultUsdtCnyRate         = 6.4      // 默认USDT汇率
	DefaultTrxCnyRate          = 0.95     // 默认TRX汇率
	defaultAuthToken           = "123234" // 默认授权码
	defaultListen              = ":8080"  // 默认监听地址
	defaultPaymentMinAmount    = 0.01
	defaultPaymentMaxAmount    = 99999
	defaultUsdtAtomicity       = 0.01 // 原子精度
	defaultTrxAtomicity        = 0.01
	defaultTronGrpcNode        = "18.141.79.38:50051"                 // 默认GRPC节点
	defaultBscRpcEndpoint      = "https://bsc-dataseed.bnbchain.org/" // 默认BSC RPC节点
	defaultPolygonRpcEndpoint  = "https://polygon-rpc.com/"           // 默认Polygon RPC节点
	defaultEthereumRpcEndpoint = "https://ethereum.publicnode.com/"   // 默认Ethereum RPC节点
	defaultOutputLog           = "/var/log/bepusdt.log"               // 默认日志输出文件
	defaultSqlitePath          = "/var/lib/bepusdt/sqlite.db"         // 默认数据库文件
)
