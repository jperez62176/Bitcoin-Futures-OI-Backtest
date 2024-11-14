package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"io"
	"log"
	
)


func GetHistoricalDataFromMarketAPI(tokenId string) (*Dataset, error) {
	url := "http://localhost:3000/token/data/" + tokenId // this is going to have to change when i deploy
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode != 200 {
		return nil, errors.New("something went wrong pinging CoinGecko API")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var dataset Dataset
	if err := json.Unmarshal(body, &dataset); err != nil {
		log.Println("Could not parse the body")
		return nil, err
	}
	return &dataset, nil

}