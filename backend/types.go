package main

import (
	"errors"
	"log"
	"time"

	"github.com/jperez62176/bitcoin-open-interest-backtest/ta"
)

type Dataset struct {
	Date  []time.Time `json:"date"`
	Price []float64   `json:"price"`
}

type TpiDatapoint struct {
	Date time.Time `json:"date"`
	Ltpi float64   `json:"ltpi"`
	Mtpi float64   `json:"mtpi"`
}

type AssetDatapoint struct {
	Date  time.Time `json:"time"`
	Price float64   `json:"price"`
}

type Response struct {
	AssetData                 Dataset           `json:"assetData"`
	LeverageData              []*AssetDatapoint `json:"leverageData"`
	TpiData                   []*TpiDatapoint   `json:"tpiData"`
	StrategyInfo              StrategyInfo      `json:"strategyInfo"`
	BitcoinFuturesOIStratInfo StrategyInfo      `json:"btcFuturesOI"`
}

type Trade struct {
	Date   time.Time `json:"date"`
	Action string    `json:"action"`
}

type StrategyInfo struct {
	Equity      []float64 `json:"equity"`
	Spot        []float64 `json:"spot"`
	Leverage    []float64 `json:"leverage"`
	Trades      []Trade   `json:"trades"`
	TotalTrades uint32    `json:"totalTrades"`
	Sharpie     float64   `json:"sharpie"`
	Sortino     float64   `json:"sortino"`
	Omega       float64   `json:"omega"`
	MaxDrawdown float32   `json:"maxDrawdown"`
	NetProfit   float64   `json:"netProfit"`
}

type BitcoinFuturesOI struct {
	URL                       string    `bson:"_id"`
	Date                      []string  `bson:"Date"`
	Plus1sd                   []float64 `bson:"+1sd"`
	Plus2sd                   []float64 `bson:"+2sd"`
	Minus1sd                  []float64 `bson:"-1sd"`
	Minus2sd                  []float64 `bson:"-2sd"`
	Deleveraging              []float64 `bson:"Deleveraging"`
	FuturesOIDayChangePercent []float64 `bson:"Futures OI 1-day Change (%)"`
	FutureOIBTC               []float64 `bson:"Futures Open Interest [BTC]"`
	LeverageHigh              []float64 `bson:"Leverage High"`
	BtcPrice                  []float64 `bson:"Price"`
}

func (strategy *StrategyInfo) CalculateStrategyMetrics() (error) {
	if strategy.Equity == nil || !(len(strategy.Equity) > 1) {
		return errors.New("no equity to calculate metrics from.")
	}

	equity := strategy.Equity
	strategy.MaxDrawdown = ta.EquityMaxDrawdown(equity)
	strategy.Sharpie = ta.SharpieRatio(equity, 365)
	strategy.Sortino = ta.SortinoRatio(strategy.Equity, 365)
	strategy.Omega = ta.OmegaRatio(strategy.Equity)
	strategy.NetProfit = equity[0] - equity[len(equity)-1]/equity[len(equity)-1]
	strategy.TotalTrades = uint32(len(strategy.Trades))
	return nil
}

// Resizes all the lists into the given length to match other datasets lengths.
func (b BitcoinFuturesOI) ResizeAllArraysTo(length int) {
	if len(b.Date) > length {
		log.Printf("inside if statement: length equals: %d", length)
		b.Date = b.Date[(len(b.Date) - length):]
		b.BtcPrice = b.BtcPrice[len(b.BtcPrice)-length:]
		b.Deleveraging = b.Deleveraging[len(b.Deleveraging)-length:]
		b.FutureOIBTC = b.FutureOIBTC[len(b.FutureOIBTC)-length:]
		b.FuturesOIDayChangePercent = b.FuturesOIDayChangePercent[len(b.FuturesOIDayChangePercent)-length:]
		b.LeverageHigh = b.LeverageHigh[len(b.LeverageHigh)-length:]
		b.Plus1sd = b.Plus1sd[len(b.Plus1sd)-length:]
		b.Plus2sd = b.Plus2sd[len(b.Plus2sd)-length:]
		b.Minus1sd = b.Minus1sd[len(b.Minus1sd)-length:]
		b.Minus2sd = b.Minus2sd[len(b.Minus2sd)-length:]
	}
}
