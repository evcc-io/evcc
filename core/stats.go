package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
)

// publisher gives access to the site's publish function
type publisher interface {
	publish(key string, val interface{})
}

// Publishes long term charging statistics
type Stats struct {
	updated time.Time // Time of last charged value update
	log     *util.Logger
}

func NewStats() *Stats {
	return &Stats{
		log: util.NewLogger("stats"),
	}
}

// Update publishes stats for the last 30 and 365 days
func (s *Stats) Update(p publisher) {
	if time.Since(s.updated) < time.Hour {
		return
	}

	s.publish(30, p)
	s.publish(365, p)

	s.updated = time.Now()
}

// publish calculates and publishes stats for the last n-days
func (s *Stats) publish(days int, p publisher) {
	// update every hour
	if time.Since(s.updated) < time.Hour {
		return
	}

	if db.Instance == nil {
		return
	}

	// Calculate the date from numberOfDays ago
	fromDate := time.Now().AddDate(0, 0, -days)

	// Struct to hold the results
	var result struct {
		SolarPercentage float64
		ChargedKWh      float64
		AvgPrice        float64
	}

	// First query to calculate solar_percentage and total_kwh
	err := db.Instance.Raw(`
		SELECT SUM(charged_kwh * (solar_percentage / 100)) / SUM(charged_kwh) AS SolarPercentage, 
			SUM(charged_kwh) as ChargedKWh 
		FROM sessions 
		WHERE finished >= ? 
		AND charged_kwh > 0 
		AND solar_percentage IS NOT NULL`, fromDate).Scan(&result).Error
	if err != nil {
		s.log.ERROR.Printf("error executing solar stats query: %v", err)
	}

	// Second query to calculate avg_price
	err = db.Instance.Raw(`
		SELECT SUM(charged_kwh * price_per_kwh) / SUM(charged_kwh) AS AvgPrice 
		FROM sessions 
		WHERE finished >= ? 
		AND charged_kwh > 0 
		AND price_per_kwh IS NOT NULL`, fromDate).Scan(&result).Error
	if err != nil {
		s.log.ERROR.Printf("error executing price stats query: %v", err)
	}

	prefix := fmt.Sprintf("stats%d", days)
	p.publish(prefix+"SolarPercentage", result.SolarPercentage)
	p.publish(prefix+"ChargedKWh", result.ChargedKWh)
	p.publish(prefix+"AvgPrice", result.AvgPrice)
}
