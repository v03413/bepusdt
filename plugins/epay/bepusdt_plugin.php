<?php

class bepusdt_plugin
{
    static public $info = [
        'name'     => 'bepusdt',
        'showname' => 'Bepusdt(一款更好用的个人USDT收款网关)',
        'author'   => '莫名',
        'link'     => 'https://github.com/v03413/bepusdt', //支付插件作者链接
        'types'    => ['usdt'], //支付插件支持的支付方式，可选的有alipay,qqpay,wxpay,bank
        'inputs'   => [ //支付插件要求传入的参数以及参数显示名称，可选的有appid,appkey,appsecret,appurl,appmchid
                        'appid'     => [
                            'name' => '接口地址',
                            'type' => 'input',
                            'note' => '必须以http://或https://开头，以/结尾',
                        ],
                        'appsecret' => [
                            'name' => 'Token',
                            'type' => 'input',
                            'note' => '',
                        ]
        ],
        'select'   => null,
        'note'     => ''
    ];

    public static function submit(): array
    {
        global $siteurl, $channel, $order, $conf;

        $data = [
            "order_id"     => TRADE_NO,
            "amount"       => (float)$order['realmoney'],
            "notify_url"   => $conf['localurl'] . 'pay/notify/' . TRADE_NO . '/',
            "redirect_url" => $siteurl . 'pay/return/' . TRADE_NO . '/',
        ];
        $resp = self::createTransaction($data, $channel['appid'], $channel['appsecret']);
        $json = json_decode($resp, true);
        if (!is_array($json) || $json['status_code'] != 200) {

            return ['type' => 'error', 'msg' => 'Bepusdt 订单创建失败，请检测相关配置是否错误！'];
        }

        return ['type' => 'jump', 'url' => $json['data']['payment_url']];
    }

    public static function mapi()
    {
        global $siteurl, $channel, $order, $conf, $device, $mdevice;

        if ($channel['appswitch'] == 1) {
            $typename = $order['typename'];
            return self::$typename();
        } else {
            return ['type' => 'jump', 'url' => $siteurl . 'pay/submit/' . TRADE_NO . '/'];
        }
    }

    public static function notify()
    {
        global $channel, $order;

        $data = json_decode(file_get_contents('php://input'), true);
        if (!is_array($data)) {

            exit('fail');
        }

        $sign = $data['signature'];
        unset($data['signature']);

        if ($sign != self::toSign($data, $channel['appsecret'])) {

            exit('sign error');
        }

        processNotify($order, $data['trade_id']);

        exit('ok');
    }

    public static function return()
    {
        global $order;

        processReturn($order, $order['api_trade_no']);
    }

    private static function toSign(array $data, $token): string
    {
        ksort($data);
        $sign = '';
        foreach ($data as $k => $v) {
            if ($v == '') continue;
            $sign .= $k . '=' . $v . '&';
        }
        $sign = trim($sign, '&');
        return md5($sign . $token);
    }

    private static function createTransaction(array $data, $api, $token)
    {
        $data['signature'] = self::toSign($data, $token);

        $url    = $api . 'api/v1/order/create-transaction';
        $header = ['Content-Type: application/json; charset=UTF-8'];
        $curl   = curl_init($url);
        curl_setopt($curl, CURLOPT_SSL_VERIFYPEER, false); //SSL证书认证false
        curl_setopt($curl, CURLOPT_SSL_VERIFYHOST, false); //严格认证false
        curl_setopt($curl, CURLOPT_HTTPHEADER, $header); //设置HTTPHEADER
        curl_setopt($curl, CURLOPT_RETURNTRANSFER, 1); // 显示输出结果
        curl_setopt($curl, CURLOPT_POST, true); // post传输数据
        curl_setopt($curl, CURLOPT_POSTFIELDS, json_encode($data)); // post传输数据
        $res = curl_exec($curl);
        curl_close($curl);
        return $res;
    }
}