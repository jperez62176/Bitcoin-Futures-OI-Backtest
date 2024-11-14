package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"strconv"
	"strings"
	"time"

	// "github.com/jperez62176/bitcoin-open-interest-backtest/ta"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/joho/godotenv"
	
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	// GETTING TPI DATA
	tpis, err := GetAdamTpiData()
	if err != nil {
		log.Fatal("Error while getting tpi data: ", err)
	}

	// GETTING ASSET DATA
	assetPrice, err := GetHistoricalDataFromMarketAPI("bitcoin")
	if err != nil {
		log.Fatal("Error while getting market api data: ", err)
	}
	if len(tpis) > len(assetPrice.Date) {
		tpis = tpis[(len(tpis) - len(assetPrice.Date)):]
	} else {
		assetPrice.Date = assetPrice.Date[len(assetPrice.Date)-len(tpis):]
		assetPrice.Price = assetPrice.Price[len(assetPrice.Price)-len(tpis):]
	}

	// GETTING LEVERAGE DATA
	leveragePrice, err := GetCsvLeverageData("BTC3XPOL.csv")
	if err != nil {
		log.Fatal("Error while getting leverage data: ", err)
	}
	if len(leveragePrice) > len(assetPrice.Date) {
		leveragePrice = leveragePrice[(len(leveragePrice) - len(assetPrice.Date)):]
	} else {
		assetPrice.Date = assetPrice.Date[len(assetPrice.Date)-len(leveragePrice):]
		assetPrice.Price = assetPrice.Price[len(assetPrice.Price)-len(leveragePrice):]
		tpis = tpis[len(tpis)-len(leveragePrice):]
	}

	// GETTING BITCOIN FUTURES OI
	store, err := GetStorage()
	if err != nil {
		log.Fatalln("Could not get mongo storage")
	}
	btcOpenInterest, err := store.GetBTCFuturesOIData("https://charts.checkonchain.com/btconchain/derivatives/derivatives_futuresoi_1daychange/derivatives_futuresoi_1daychange_light.html")
	if err != nil {
		log.Fatal("Error while getting bitcoin futures oi data: ", err)
	}

	btcOpenInterest.ResizeAllArraysTo(len(leveragePrice))
	

	// CALCULATING STRATEGY DATA
	strategyInfo := TPIsJointStrategyEquityCurve(tpis, assetPrice, leveragePrice)
	rebalancingStrategyInfo := RebalancingTPIsJointStratWithBTCFuturesOICriteriaStrategyEquityCurve(tpis, assetPrice, leveragePrice, btcOpenInterest)
	strategyInfo.CalculateStrategyMetrics()
	rebalancingStrategyInfo.CalculateStrategyMetrics()

	// RENDERING WEBSITE
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("plotly.html")
		if err != nil {
			log.Fatalf("template parsing error: %v", err)
		}

		jsonData, err := json.Marshal(Response{AssetData: *assetPrice, TpiData: tpis, StrategyInfo: *strategyInfo, LeverageData: leveragePrice, BitcoinFuturesOIStratInfo: *rebalancingStrategyInfo})

		if err != nil {
			log.Fatalf("JSON marshaling error: %v", err)
		}

		// Pass JSON data to the template
		tmpl.Execute(w, string(jsonData))
	})

	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func GetBTCFuturesOIJsonData(filename string) (*BitcoinFuturesOI, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var btcFuturesOI BitcoinFuturesOI
	err = json.Unmarshal(data, &btcFuturesOI)
	if err != nil {
		return nil, err
	}
	log.Println(btcFuturesOI)
	return &btcFuturesOI, nil
}

func GetCsvLeverageData(filename string) ([]*AssetDatapoint, error) {
	// Open the CSV file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
		return nil, err
	}
	defer file.Close()
	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
		return nil, err
	}

	// Print each record
	datapoints := make([]*AssetDatapoint, 0)
	for _, record := range records {
		date, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, err
		}

		price, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, err
		}
		datapoints = append(datapoints, &AssetDatapoint{Date: date, Price: price})
	}
	return datapoints, nil
}

func GetAdamTpiData() ([]*TpiDatapoint, error) {
	ctx := context.Background()
	adamsTpiSpreadsheetId := "1fYu3iVWrBxZORbWg0wKvqMWPdGsFOVDQ4KRIy1GL1XE"
	// Set the credentials
	client, err := sheets.NewService(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		fmt.Printf("Error creating Sheets service: %v\n", err)
		return nil, err
	}

	readRange := "A3:C544"
	resp, err := client.Spreadsheets.Values.Get(adamsTpiSpreadsheetId, readRange).Do()
	if err != nil {
		fmt.Printf("Unable to retrieve data from sheet: %v\n", err)
		return nil, err
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
		return nil, err
	}

	datapoints := make([]*TpiDatapoint, 0)
	// Print the values
	for _, row := range resp.Values {

		var date time.Time
		var ltpi, mtpi float64
		for i, cell := range row {
			switch i {
			case 0:
				dateStr := insertCharAtIndex(strings.ReplaceAll(cell.(string), ".", "-"), "20", 6)
				date, err = time.Parse("02-01-2006", dateStr)
				if err != nil {
					return nil, err
				}
			case 1:
				ltpi, err = strconv.ParseFloat(cell.(string), 64)
				if err != nil {
					return nil, err
				}
			case 2:
				mtpi, err = strconv.ParseFloat(cell.(string), 64)
				if err != nil {
					return nil, err
				}
			}
		}

		datapoints = append(datapoints, &TpiDatapoint{Date: date, Ltpi: ltpi, Mtpi: mtpi})
	}
	log.Println("Created the datapoints successfully. List length: ", len(datapoints))
	return datapoints, nil
}

func insertCharAtIndex(str string, char string, index int) string {
	return str[:index] + char + str[index:]
}

// Create a line chart
func TPIsJointStrategyEquityCurve(tpiDataset []*TpiDatapoint, assetDataset *Dataset, leverageDataset []*AssetDatapoint) *StrategyInfo {
	strategyInfo := new(StrategyInfo)
	totalDatapoints := len(tpiDataset)
	trades := make([]Trade, 0)

	var totalPortfolio float64 = 1
	var assetPosition float64 = 1
	var leveragePosition float64 = 0
	var leverageProportion = 0.3
	var prevAssetValue float64
	var prevLeverageValue float64
	var position = "none"

	leverage := false
	cash := true
	// startSimulation := false
	for index := 0; index < totalDatapoints; index++ {
		assetValue := assetDataset.Price[index]
		leverageValue := leverageDataset[index].Price
		Ltpi := tpiDataset[index].Ltpi
		Mtpi := tpiDataset[index].Mtpi
		// fmt.Println("LTPI:", Ltpi,", MTPI: ", Mtpi )
		if position == "leverage permissable" {
			assetPosition = assetPosition + ((assetValue - prevAssetValue) / prevAssetValue * assetPosition)
			// fmt.Println("LP ", leveragePosition, ", LV ", leverageValue, ", pLV ", prevLeverageValue)
			leveragePosition = leveragePosition + ((leverageValue - prevLeverageValue) / prevLeverageValue * leveragePosition)
		} else if position == "spot only" {
			assetPosition = assetPosition + ((assetValue - prevAssetValue) / prevAssetValue * assetPosition)
		}

		if Ltpi < -0.1 && Mtpi < -0.1 {
			if !cash {
				position = "none"
				cash = true
				strategyInfo.Trades = append(trades, Trade{Action: position, Date: leverageDataset[index].Date})
			}
		} else if Ltpi > 0.1 && Mtpi > 0.1 {
			if !leverage {
				position = "leverage permissable"
				cash = false
				leverage = true

				assetPosition = assetPosition + leveragePosition
				leveragePosition = assetPosition * leverageProportion
				assetPosition = assetPosition - leveragePosition
				strategyInfo.Trades = append(trades, Trade{Action: position, Date: leverageDataset[index].Date})
			}
		} else if Ltpi > 0.1 || Mtpi > 0.1 {
			if leverage || cash {
				position = "spot only"
				cash = false
				leverage = false
				assetPosition = assetPosition + leveragePosition
				leveragePosition = 0
				strategyInfo.Trades = append(trades, Trade{Action: position, Date: leverageDataset[index].Date})
			}
		}

		// fmt.Println(leveragePosition)
		// if !startSimulation && currentRsi != 0 {
		// 	startSimulation = true
		// }

		prevLeverageValue = leverageValue
		prevAssetValue = assetValue
		totalPortfolio = assetPosition + leveragePosition
		//fmt.Println("Total position: ", totalPortfolio, ", Asset position:", assetPosition, ", Leverage Position: ", leveragePosition)
		strategyInfo.Equity = append(strategyInfo.Equity, totalPortfolio)
		strategyInfo.Spot = append(strategyInfo.Spot, assetPosition)
		strategyInfo.Leverage = append(strategyInfo.Leverage, leveragePosition)

	}

	return strategyInfo
}

func RebalancingTPIsJointStratWithBTCFuturesOICriteriaStrategyEquityCurve(tpiDataset []*TpiDatapoint, assetDataset *Dataset, leverageDataset []*AssetDatapoint, btcFutureOI *BitcoinFuturesOI) *StrategyInfo {
	strategyInfo := new(StrategyInfo)
	totalDatapoints := len(tpiDataset)
	trades := make([]Trade, 0)

	var totalPortfolio float64 = 1
	var assetPosition float64 = 1
	var leveragePosition float64 = 0
	var leverageProportion = 0.3
	var prevAssetValue float64
	var prevLeverageValue float64
	var position = "none"

	leverage := false
	cash := true
	// startSimulation := false

	percentFutureOI := btcFutureOI.FuturesOIDayChangePercent

	log.Printf("Percent FOI: %d, total Datapoints: %d", len(percentFutureOI), totalDatapoints)

	for index := 0; index < totalDatapoints; index++ {
		assetValue := assetDataset.Price[index]
		leverageValue := leverageDataset[index].Price
		Ltpi := tpiDataset[index].Ltpi
		Mtpi := tpiDataset[index].Mtpi
		// fmt.Println("LTPI:", Ltpi,", MTPI: ", Mtpi )
		if position == "leverage permissable" {
			assetPosition = assetPosition + ((assetValue - prevAssetValue) / prevAssetValue * assetPosition)
			// fmt.Println("LP ", leveragePosition, ", LV ", leverageValue, ", pLV ", prevLeverageValue)
			leveragePosition = leveragePosition + ((leverageValue - prevLeverageValue) / prevLeverageValue * leveragePosition)
		} else if position == "spot only" {
			assetPosition = assetPosition + ((assetValue - prevAssetValue) / prevAssetValue * assetPosition)
		}

		if Ltpi < -0.1 && Mtpi < -0.1 {
			if !cash {
				position = "none"
				cash = true
				strategyInfo.Trades = append(trades, Trade{Action: position, Date: leverageDataset[index].Date})
			}
		} else if Ltpi > 0.1 && Mtpi > 0.1 {
			if !leverage {
				position = "leverage permissable"
				cash = false
				leverage = true

				assetPosition = assetPosition + leveragePosition
				leveragePosition = assetPosition * leverageProportion
				assetPosition = assetPosition - leveragePosition
				strategyInfo.Trades = append(trades, Trade{Action: position, Date: leverageDataset[index].Date})
			} else if leverage && percentFutureOI[index] > btcFutureOI.Plus1sd[index] { // REBALANCING
				assetPosition += leveragePosition      // 100% spot
				leveragePosition = assetPosition * 0.3 // 30% Leverage
				assetPosition -= leveragePosition      // 70% spot
				strategyInfo.Trades = append(trades, Trade{Action: "rebalancing", Date: leverageDataset[index].Date})
			}
		} else if Ltpi > 0.1 || Mtpi > 0.1 {
			if leverage || cash {
				position = "spot only"
				cash = false
				leverage = false
				assetPosition = assetPosition + leveragePosition
				leveragePosition = 0
				strategyInfo.Trades = append(trades, Trade{Action: position, Date: leverageDataset[index].Date})
			}
		}

		// fmt.Println(leveragePosition)
		// if !startSimulation && currentRsi != 0 {
		// 	startSimulation = true
		// }

		prevLeverageValue = leverageValue
		prevAssetValue = assetValue
		totalPortfolio = assetPosition + leveragePosition
		fmt.Println("Total position: ", totalPortfolio, ", Asset position:", assetPosition, ", Leverage Position: ", leveragePosition)
		strategyInfo.Equity = append(strategyInfo.Equity, totalPortfolio)
		strategyInfo.Spot = append(strategyInfo.Spot, assetPosition)
		strategyInfo.Leverage = append(strategyInfo.Leverage, leveragePosition)

	}

	return strategyInfo
}

