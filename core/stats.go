package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/core/keys"
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
		"30d":      s.calculate(30),
		"365d":     s.calculate(365),
		"thisYear": s.calculate(time.Now().YearDay()),
		"total":    s.calculate(365 * 100), // 100 years
	}
	p.publish(keys.Statistics, stats)

	s.updated = time.Now()
}

// calculate reads the stats for the last n-days
func (s *Stats) calculate(days int) map[string]float64 {
	result := make(map[string]float64)

	executeQuery := func(selectClause string, whereClause string, fromDate time.Time, dest interface{}) {
		query := fmt.Sprintf(`
		SELECT COALESCE(%s, 0)
		FROM sessions
		WHERE finished >= ? 
		AND charged_kwh > 0 
		%s`, selectClause, whereClause)

		if err := db.Instance.Raw(query, fromDate).Scan(dest).Error; err != nil {
			s.log.ERROR.Printf("error executing query: %v", err)
		}
	}

	fromDate := time.Now().AddDate(0, 0, -days)
	var solarPercentage, chargedKWh, avgPrice, avgCo2 float64
	executeQuery("SUM(charged_kwh * solar_percentage) / SUM(charged_kwh)", "AND solar_percentage IS NOT NULL", fromDate, &solarPercentage)
	executeQuery("SUM(charged_kwh)", "AND solar_percentage IS NOT NULL", fromDate, &chargedKWh)
	executeQuery("SUM(charged_kwh * price_per_kwh) / SUM(charged_kwh)", "AND price_per_kwh IS NOT NULL", fromDate, &avgPrice)
	executeQuery("SUM(charged_kwh * co2_per_kwh) / SUM(charged_kwh)", "AND co2_per_kwh IS NOT NULL", fromDate, &avgCo2)

	result["solarPercentage"] = solarPercentage
	result["chargedKWh"] = chargedKWh
	result["avgPrice"] = avgPrice
	result["avgCo2"] = avgCo2

	return result
}
