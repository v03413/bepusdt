package model

const installSql = `
-- trade_orders definition

CREATE TABLE trade_orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL, -- 客户订单ID
	trade_id TEXT UNIQUE NOT NULL, -- 本地订单ID
    trade_hash TEXT NULL, -- 交易哈希
	usdt_rate TEXT NOT NULL, -- USDT汇率
    amount TEXT NOT NULL, -- USDT交易数额
    money TEXT NOT NULL, -- 订单交易金额
    address TEXT NOT NULL, -- 收款地址
    from_address TEXT NOT NULL, -- 支付地址
    status INTEGER DEFAULT 1 NOT NULL, -- 交易状态 1：等待支付 2：支付成功 3：订单过期
    return_url TEXT NULL, -- 同步地址
    notify_url TEXT NOT NULL, -- 异步地址
    notify_num INTEGER DEFAULT 0, -- 回调次数
    notify_state INTEGER DEFAULT 0, -- 回调状态 1：成功 0：失败
    expired_at TIMESTAMP NOT NULL,  -- 订单失效时间
	created_at TIMESTAMP NOT NULL,  -- 订单创建时间
	confirmed_at TIMESTAMP NULL, -- 交易确认时间
    updated_at TIMESTAMP NULL   -- 最后更新时间
);

-- wallet_address definition

CREATE TABLE wallet_address (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    address TEXT UNIQUE NOT NULL, -- 钱包地址
    status INTEGER DEFAULT 1 NOT NULL, -- 地址状态 1启动 0禁止
	other_notify INTEGER DEFAULT 1 NOT NULL, -- 其它转账通知 1启动 0禁止
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL
);

-- notify_record definition

CREATE TABLE notify_record (
    txid TEXT PRIMARY KEY NOT NULL, -- 交易哈希
	created_at TIMESTAMP NOT NULL,  -- 创建时间
    updated_at TIMESTAMP NULL   -- 更新时间
);
`
