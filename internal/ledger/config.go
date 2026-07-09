package ledger

import (
	_ "embed"
)

//go:embed prelude.cue
var Prelude string

type RuntimeConfig struct {
	OperatingCurrency []string `json:"operating_currency"`
	Commodities      map[string]CommodityConfig `json:"commodities"`
	Accounts         map[string]AccountConfig `json:"accounts"`
}

type CommodityConfig struct {
	Precision int `json:"precision"`
}

type AccountConfig struct {
	Opened     bool     `json:"opened"`
	Closed     bool     `json:"closed"`
	Currencies []string `json:"currencies"`
}
