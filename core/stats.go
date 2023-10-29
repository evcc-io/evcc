package core

import (
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

// Update publishes stats based on charging sessions
func (s *Stats) Update(p publisher) {
	if time.Since(s.updated) < time.Hour {
		return
	}

	stats := map[string]map[string]float64{
		"30d":   s.calculate(30),
		"365d":  s.calculate(365),
		"total": s.calculate(365 * 100), // 100 years
	}
	p.publish("statistics", stats)

	s.updated = time.Now()
}

// calculate reads the stats for the last n-days
func (s *Stats) calculate(days int) map[string]float64 {
	result := make(map[string]float64)

	// Calculate the date from numberOfDays ago
	fromDate := time.Now().AddDate(0, 0, -days)

	// Struct to hold the results
	var dbResult struct {
		SolarPercentage float64
		ChargedKWh      float64
		AvgPrice        float64
		AvgCo2          float64
	}

	// Calculate solar_percentage and total_kwh
	if err := db.Instance.Raw(`
		SELECT SUM(charged_kwh * solar_percentage) / SUM(charged_kwh) AS SolarPercentage, 
			SUM(charged_kwh) as ChargedKWh 
		FROM sessions 
		WHERE finished >= ? 
		AND charged_kwh > 0 
		AND solar_percentage IS NOT NULL`, fromDate).Scan(&dbResult).Error; err != nil {
		s.log.ERROR.Printf("error executing solar stats query: %v", err)
	}

	// Calculate avg_price
	if err := db.Instance.Raw(`
		SELECT SUM(charged_kwh * price_per_kwh) / SUM(charged_kwh) AS AvgPrice 
		FROM sessions 
		WHERE finished >= ? 
		AND charged_kwh > 0 
		AND price_per_kwh IS NOT NULL`, fromDate).Scan(&dbResult).Error; err != nil {
		s.log.ERROR.Printf("error executing price stats query: %v", err)
	}

	// Calculate avg_co2
	if err := db.Instance.Raw(`
		SELECT SUM(charged_kwh * co2_per_kwh) / SUM(charged_kwh) AS AvgCo2
		FROM sessions
		WHERE finished >= ?
		AND charged_kwh > 0
		AND co2_per_kwh IS NOT NULL`, fromDate).Scan(&dbResult).Error; err != nil {
		s.log.ERROR.Printf("error executing co2 stats query: %v", err)
	}

	result["solarPercentage"] = dbResult.SolarPercentage
	result["chargedKWh"] = dbResult.ChargedKWh
	result["avgPrice"] = dbResult.AvgPrice
	result["avgCo2"] = dbResult.AvgCo2

	return result
}
