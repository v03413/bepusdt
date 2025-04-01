# 签名算法

签名生成的通用步骤如下：

第一步，将所有非空参数值的参数按照参数名ASCII码从小到大排序（字典序），使用URL键值对的格式（即key1=value1&key2=value2…）拼接成
`待加密参数`。

重要规则：   
◆ 参数名ASCII码从小到大排序（字典序）；         
◆ 如果参数的值为空不参与签名；        
◆ 参数名区分大小写；
第二步，`待加密参数`最后拼接上`api接口认证token`得到`待签名字符串`，并对`待签名字符串`进行MD5运算，再将得到的`MD5字符串`
所有字符转换为`小写`，得到签名`signature`。 注意：`signature`的长度为32个字节。

举例：

假设传送的参数如下：

```
order_id : 20220201030210321
amount : 42
notify_url : http://example.com/notify
redirect_url : http://example.com/redirect
```

假设api接口认证token为：`epusdt_password_xasddawqe`(api接口认证token可以在`conf.toml`文件设置)

第一步：对参数按照key=value的格式，并按照参数名ASCII字典序排序如下：

```
amount=42&notify_url=http://example.com/notify&order_id=20220201030210321&redirect_url=http://example.com/redirect
```

第二步：拼接API密钥并加密：

```
MD5(amount=42&notify_url=http://example.com/notify&order_id=20220201030210321&redirect_url=http://example.com/redirectepusdt_password_xasddawqe)
```

最终得到最终发送的数据：

```
order_id : 20220201030210321
amount : 42
notify_url : http://example.com/notify
redirect_url : http://example.com/redirect
signature : 1cd4b52df5587cfb1968b0c0c6e156cd
```

## 参考引用

- https://github.com/assimon/epusdt/blob/master/wiki/API.md