---
description: "中国用户的 commodities / 行情 provider 配置示范:基金、A 股、港股 / 美股、加密货币如何接上价格源,避免 XIRR 显示为 0。"
---

# 中国用户的 commodities 配置示范 <sub>:flag_cn:</sub>

如果你导入了一个真实的账本(基金、A 股、港股 / 美股、加密货币),但
打开 `资产 (Assets)` :material-chevron-right: `净值 (Net Worth)`
后发现 **XIRR 显示 `0.00`**、`收益 (Gain)` 全部按成本计价、`配置
(Allocation)` 看不到真实市值占比,通常**不是 bug**,而是因为你的
`paisa.yaml` 里没有给这些 commodity 注册 `commodities:` +
`price.provider`。没有 provider,`paisa update` 就抓不到任何行情,
持仓的"市值"等于"成本基础",于是总现金流相加为 0,XIRR 自然算出 0。

解决办法是为账本里用到的每一个基金 / 股票 / 加密货币,在配置里加上一
条 commodity,告诉 Paisa **去哪里抓价格**。

!!! tip "先确认问题"

    更新价格后,可以用下面的命令查看实际抓到了多少条价格。如果返回
    `0` 或非常少,说明 commodity 还没配好 provider:

    ```shell
    curl -s http://localhost:7500/api/price | jq '.prices | length'
    ```

## commodity 配置长什么样

每条 commodity 由四个字段组成,和 [Commodities 参考文档](../reference/commodities.md)
里完全一致:

```yaml
commodities:
  - name: HS300 # (1)!
    type: mutualfund # (2)!
    price:
        provider: cn-ttjj # (3)!
        code: "000311" # (4)!
```

1. `name` —— 必须和你 journal 里使用的 commodity 名称完全一致
1. `type` —— `mutualfund`、`stock` 或 `unknown`
1. `price.provider` —— 行情 provider 的代码(见下表)
1. `price.code` —— provider 用来定位这只标的的代码

## 常见 ticker → provider 映射

下表只列出当前代码库里**确实存在**的 provider。

| 资产类型 | 例子 | provider | type | price.code 写法 |
|----------|------|----------|------|-----------------|
| 场内 / 场外基金(天天基金) | 景顺长城沪深 300 增强 A | `cn-ttjj` | `mutualfund` | 6 位基金代码,如 `"000311"`(保留前导零,用引号) |
| A 股 | 浦发银行 `600000` | `cn-eastmoney` | `stock` | A 股代码,如 `600000`、`000001`、`300750`;也可写 `市场.代码`,如 `116.00700`、`105.AAPL` |
| 港股 / 美股 | 腾讯 `0700.HK`、Uber `UBER` | `yahoo` | `stock` | Yahoo Finance ticker,如 `0700.HK`、`1810.HK`、`AAPL`、`UBER`、`BABA` |
| 加密货币 | 比特币 | `cn-okx` | `unknown` | OKX instrument id,如 `BTC-USDT`、`ETH-USDT`、`BTC-USD` |

外汇(把外币持仓折算回本币)由 FX 层处理,默认使用 `cn-boc`(中国
银行经 frankfurter.app)和 `yahoo-fx`,通常无需手动为每只标的配置。

## 可直接复制的示例

下面是一份覆盖四类资产的完整 `commodities:` 配置示范。把 `name` 改成
你 journal 里实际用到的 commodity 名,把 `code` 改成你的标的代码即可。

```yaml
commodities:
  # 场外基金 —— 天天基金 (Eastmoney) 历史净值
  - name: HS300 # 景顺长城沪深 300 增强 A
    type: mutualfund
    price:
        provider: cn-ttjj
        code: "000311" # 6 位基金代码,保留前导零

  # A 股 —— 东方财富
  - name: SPDB # 浦发银行
    type: stock
    price:
        provider: cn-eastmoney
        code: "600000"

  # 港股 —— Yahoo Finance
  - name: TENCENT # 腾讯控股
    type: stock
    price:
        provider: yahoo
        code: 0700.HK

  # 美股 —— Yahoo Finance
  - name: UBER
    type: stock
    price:
        provider: yahoo
        code: UBER

  # 加密货币 —— OKX
  - name: BTC
    type: unknown
    price:
        provider: cn-okx
        code: BTC-USDT
```

!!! note "加密货币的计价货币"

    OKX 返回的价格以该 instrument 的报价货币计(通常是 `USDT` 或
    `USD`)。折算到你的 `default_currency` 由 FX 层完成,所以需要保证
    报价货币到默认货币之间有可用的汇率。详见
    [OKX provider 参考](../reference/providers/cn-okx.md)。

## 接好之后

1. 保存配置后,从右上角下拉菜单点击 `更新价格 (Update Prices)`,或在命令行
   运行 `paisa update`。
2. 再次运行上面的 `curl ... | jq '.prices | length'`,数字应该明显大于 0。
3. 回到 `资产 (Assets)` :material-chevron-right: `余额 (Balance)`
   查看最新市值,在 `账本 (Ledger)` :material-chevron-right:
   `价格 (Price)` 查看完整的历史价格曲线,XIRR / Gain / Allocation
   也会随之恢复正常。

## 延伸阅读

- [Commodities 参考文档](../reference/commodities.md) —— 全部 provider 与字段说明
- [天天基金 (Eastmoney) provider](../reference/providers/cn-ttjj.md)
- [东方财富 (Eastmoney) provider](../reference/providers/cn-eastmoney.md)
- [Yahoo Finance provider](../reference/providers/yahoo.md)
