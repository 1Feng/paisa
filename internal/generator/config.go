package generator

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// MinimalConfig writes a bare-bones paisa.yaml + empty journal file
// into cwd. Used by `paisa serve` on first run when no config exists.
func MinimalConfig(cwd string) {
	configFilePath := filepath.Join(cwd, "paisa.yaml")
	config := `
journal_path: '%s'
db_path: '%s'

# Register a commodity for every fund / stock / crypto used in your
# journal so Paisa can fetch prices. Without a price.provider the
# holding is valued at cost basis and XIRR shows 0. Uncomment and
# adjust the names/codes below. See https://paisa.fyi for details.
#
# commodities:
#   - name: HS300            # 场外基金 — 天天基金 (Eastmoney)
#     type: mutualfund
#     price:
#       provider: cn-ttjj
#       code: "000311"       # 6-digit fund code, keep leading zeros
#   - name: SPDB             # A 股 — 东方财富
#     type: stock
#     price:
#       provider: cn-eastmoney
#       code: "600000"
#   - name: TENCENT          # 港股 / 美股 — Yahoo Finance
#     type: stock
#     price:
#       provider: yahoo
#       code: 0700.HK
#   - name: BTC              # 加密货币 — OKX
#     type: unknown
#     price:
#       provider: cn-okx
#       code: BTC-USDT
`
	log.Info("Generating config file: ", configFilePath)
	journalFilePath := filepath.Join(cwd, "main.ledger")
	dbFilePath := filepath.Join(cwd, "paisa.db")
	err := os.WriteFile(configFilePath, []byte(fmt.Sprintf(config, filepath.Base(journalFilePath), filepath.Base(dbFilePath))), 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Generating journal file: ", journalFilePath)
	_, err = os.OpenFile(journalFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// Demo previously generated an India-specific sample journal +
// portfolio (NPS / mutual-fund / Schedule AL). That data set was
// retired together with the India business logic in M2-A. The new
// CN demo will be reintroduced in a later milestone; until then,
// Demo behaves identically to MinimalConfig so `paisa init` and the
// /api/init endpoint still produce a valid empty workspace.
func Demo(cwd string) {
	MinimalConfig(cwd)
}
