package ta

import (
	"log"
	"math"
)

func Average(data []float64) float64 {
	var sum float64
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func StandardDeviation(data []float64) float64 {
	mean := Average(data)
	var sumSquares float64
	for _, value := range data {
		sumSquares += math.Pow(value-mean, 2)
	}
	variance := sumSquares / float64(len(data))
	return math.Sqrt(variance)
}

func Crossover(currentValue float64, previousValue float64, crossOverPoint float64) bool {
	return currentValue > crossOverPoint && previousValue <= crossOverPoint
}

func Crossunder(currentValue float64, previousValue float64, crossUnderPoint float64) bool {
	return currentValue < crossUnderPoint && previousValue >= crossUnderPoint
}

func Rsi(assetDataList []float64, rsiLength int) []float64 {
	// Initialize rsiDataList with the same length as assetDataList
	rsiDataList := make([]float64, len(assetDataList))

	var gainSum, loseSum float64
	for index := 1; index <= rsiLength; index++ {
		change := assetDataList[index] - assetDataList[index-1]
		if change > 0 {
			gainSum += change
		} else {
			loseSum -= change
		}
	}

	avgGain := gainSum / float64(rsiLength)
	avgLoss := loseSum / float64(rsiLength)

	var rs, rsi float64
	if avgLoss == 0 {
		rs = math.Inf(1)
	} else {
		rs = avgGain / avgLoss
		rsi = 100 - (100 / (1 + rs))
	}
	rsiDataList[rsiLength] = rsi

	// Now calculate the rest of the RSI values using the RMA (exponential smoothing)
	for index := rsiLength + 1; index < len(assetDataList); index++ {
		change := assetDataList[index] - assetDataList[index-1]

		if change > 0 {
			avgGain = (avgGain*(float64(rsiLength-1)) + change) / float64(rsiLength)
			// no loss, so we only smooth avgLoss
			avgLoss = (avgLoss * float64(rsiLength-1)) / float64(rsiLength)
		} else {
			// no gain, so we only smooth avgGain
			avgGain = (avgGain * float64(rsiLength-1)) / float64(rsiLength)
			avgLoss = (avgLoss*(float64(rsiLength-1)) - change) / float64(rsiLength)
		}
		// Calculate the new RS and RSI
		if avgLoss == 0 {
			rs = math.Inf(1)
			rsi = 100
		} else {
			rs = avgGain / avgLoss
			rsi = 100 - (100 / (1 + rs))
		}

		// Store the RSI value
		rsiDataList[index] = rsi
	}

	return rsiDataList
}

func SharpieRatio(assetDataList []float64, yearlyLength float64) float64 {
	returns := make([]float64, len(assetDataList)-1)

	for i := 0; i < len(assetDataList)-1; i++ {
		returns[i] = (assetDataList[i+1] - assetDataList[i]) / assetDataList[i]
	}

	averageReturns := Average(returns)

	stdDev := StandardDeviation(returns)

	return averageReturns / stdDev * math.Sqrt(yearlyLength)
}

func SortinoRatio(assetDataList []float64, yearlyLength float64) float64 {
	returns := make([]float64, len(assetDataList)-1)
	var negativeReturns []float64

	for i := 0; i < len(assetDataList)-1; i++ {
		_return := (assetDataList[i+1] - assetDataList[i]) / assetDataList[i]
		returns[i] = _return
		if _return <= 0 {
			negativeReturns = append(negativeReturns, _return)
		}
	}
	meanReturns := Average(returns)

	negativeStandardDeviation := StandardDeviation(negativeReturns)

	return meanReturns / negativeStandardDeviation * math.Sqrt(yearlyLength)
}

func OmegaRatio(assetDataList []float64) float64 {
	var positiveReturns float64
	var negativeReturns float64
	for i := 1; i < len(assetDataList); i++ {
		_return := (assetDataList[i] - assetDataList[i-1]) / assetDataList[i-1]
		if _return <= 0 {
			negativeReturns += _return
		} else {
			positiveReturns += _return
		}
	}
	return positiveReturns / negativeReturns * -1
}

func EquityMaxDrawdown(assetData []float64) float32 {
	var peak, trough, maxDrawdown float64
	var troughMode = false
	for index := 0; index < len(assetData)-1; index++ {
		log.Print(index)
		currentValue := assetData[index]
		nextValue := assetData[index+1]
		if !troughMode && currentValue > nextValue {
			peak = currentValue
			trough = nextValue
			troughMode = true
			continue
		}
		
		lastViableIndex := index == len(assetData)-2

		if (troughMode && currentValue > peak) || lastViableIndex {
			drawdown := (trough - peak) / peak
			if drawdown < maxDrawdown {
				maxDrawdown = drawdown
			}
			if lastViableIndex {
				break
			}
			troughMode = false
			index--
		}
		if troughMode && currentValue < trough {
			trough = currentValue
		}

	}
	return float32(maxDrawdown)
}
