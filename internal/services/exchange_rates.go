package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

const frankfurterLatestURL = "https://api.frankfurter.dev/v1/latest?from=%s&to=TRY"

type LatestExchangeRate struct {
	Currency  string  `json:"currency"`
	RateToTRY float64 `json:"rate_to_try"`
	Source    string  `json:"source"`
	RateDate  string  `json:"rate_date"`
}

type FrankfurterResponse struct {
	Amount float64 `json:"amount"`
	Base   string  `json:"base"`
	Date   string  `json:"date"`
	Rates  struct {
		TRY float64 `json:"TRY"`
	} `json:"rates"`
}

func NormalizeCurrency(value string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(value))
	if currency == "" {
		return "TRY", nil
	}
	switch currency {
	case "TRY", "USD", "EUR":
		return currency, nil
	default:
		return "", errors.New("currency is invalid")
	}
}

func LatestRateToTRY(ctx context.Context, db *gorm.DB, currency string) (LatestExchangeRate, error) {
	currency, err := NormalizeCurrency(currency)
	if err != nil {
		return LatestExchangeRate{}, err
	}
	if currency == "TRY" {
		return LatestExchangeRate{Currency: "TRY", RateToTRY: 1, Source: "system", RateDate: time.Now().Format("2006-01-02")}, nil
	}

	rate, err := fetchFrankfurterRate(ctx, currency)
	if err == nil {
		if err := saveExchangeRate(db, rate); err != nil {
			// A successfully fetched rate remains usable even if the cache write fails.
			log.Printf("[ExchangeRate Error] cache save failed for %s: %v", currency, err)
		}
		return rate, nil
	}
	log.Printf("[ExchangeRate Error] live rate fetch failed for %s: %v", currency, err)

	cachedRate, cachedErr := latestStoredExchangeRate(db, currency)
	if cachedErr == nil {
		log.Printf("[ExchangeRate] using cached %s -> TRY = %v", currency, cachedRate.RateToTRY)
		return cachedRate, nil
	}
	log.Printf("[ExchangeRate Error] no cached rate for %s: %v", currency, cachedErr)
	return LatestExchangeRate{}, errors.New("exchange rate unavailable; manual exchange_rate is required")
}

func fetchFrankfurterRate(ctx context.Context, currency string) (LatestExchangeRate, error) {
	url := fmt.Sprintf(frankfurterLatestURL, currency)
	log.Printf("[ExchangeRate] request URL: %s", url)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[ExchangeRate Error] request creation failed: %v", err)
		return LatestExchangeRate{}, err
	}
	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		log.Printf("[ExchangeRate Error] request failed: %v", err)
		return LatestExchangeRate{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("[ExchangeRate Error] response read failed: %v", err)
		return LatestExchangeRate{}, err
	}
	log.Printf("[ExchangeRate] HTTP status: %d", response.StatusCode)
	log.Printf("[ExchangeRate] response body: %s", string(body))
	if response.StatusCode != http.StatusOK {
		return LatestExchangeRate{}, fmt.Errorf("exchange rate provider returned status %d", response.StatusCode)
	}

	var payload FrankfurterResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[ExchangeRate Error] response parse failed: %v", err)
		return LatestExchangeRate{}, err
	}
	rateValue := payload.Rates.TRY
	if rateValue <= 0 {
		log.Printf("[ExchangeRate Error] parsed TRY rate is invalid: %v", rateValue)
		return LatestExchangeRate{}, errors.New("exchange rate provider returned no TRY rate")
	}
	rateDate, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		log.Printf("[ExchangeRate Error] rate date parse failed: %v", err)
		return LatestExchangeRate{}, err
	}
	log.Printf("[ExchangeRate] parsed rate: %s -> TRY = %v", currency, rateValue)
	log.Printf("[ExchangeRate] %s -> TRY = %v", currency, rateValue)
	return LatestExchangeRate{
		Currency:  currency,
		RateToTRY: rateValue,
		Source:    "Frankfurter",
		RateDate:  rateDate.Format("2006-01-02"),
	}, nil
}

func saveExchangeRate(db *gorm.DB, rate LatestExchangeRate) error {
	rateDate, err := time.Parse("2006-01-02", rate.RateDate)
	if err != nil {
		return err
	}
	var stored models.ExchangeRate
	err = db.Where("currency_code = ? AND source = ? AND rate_date = ?", rate.Currency, rate.Source, rateDate).First(&stored).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&models.ExchangeRate{
			CurrencyCode: rate.Currency,
			RateToTRY:    decimal.NewFromFloat(rate.RateToTRY),
			Source:       rate.Source,
			RateDate:     rateDate,
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&stored).Update("rate_to_try", decimal.NewFromFloat(rate.RateToTRY)).Error
}

func latestStoredExchangeRate(db *gorm.DB, currency string) (LatestExchangeRate, error) {
	var stored models.ExchangeRate
	if err := db.Where("currency_code = ?", currency).Order("rate_date desc, created_at desc, id desc").First(&stored).Error; err != nil {
		return LatestExchangeRate{}, err
	}
	return LatestExchangeRate{
		Currency:  stored.CurrencyCode,
		RateToTRY: stored.RateToTRY.InexactFloat64(),
		Source:    stored.Source,
		RateDate:  stored.RateDate.Format("2006-01-02"),
	}, nil
}
