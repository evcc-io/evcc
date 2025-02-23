package openmeteo

import (
    "sort"
    "time"

)

// Estimate holds the forecast results from Open-Meteo.
type Estimate struct {
    Watts       map[time.Time]int
    WhPeriod    map[time.Time]int
    WhDays      map[time.Time]int
    ApiTimezone *time.Location
}

// NewEstimate creates a new Estimate instance.
func NewEstimate(ApiTimezone *time.Location) *Estimate {
    return &Estimate{
        Watts:       make(map[time.Time]int),
        WhPeriod:    make(map[time.Time]int),
        WhDays:      make(map[time.Time]int),
        ApiTimezone: ApiTimezone,
    }
}

// TimedValue returns the value for a specific time.
func TimedValue(at time.Time, data map[time.Time]int) int {
    var value int
    for timestamp, curValue := range data {
        if timestamp.After(at) {
            return value
        }
        value = curValue
    }
    return value
}

// IntervalValueSum returns the sum of values in the interval.
func IntervalValueSum(intervalBegin, intervalEnd time.Time, data map[time.Time]int) int {
    total := 0
    for timestamp, wh := range data {
        if timestamp.Before(intervalBegin) {
            continue
        }
        if timestamp.After(intervalEnd) {
            break
        }
        total += wh
    }
    return total
}

// IntervalValueSumWToWh converts W to Wh and returns the sum of values in the interval.
func IntervalValueSumWToWh(intervalBegin, intervalEnd time.Time, data map[time.Time]int) int {
    total := 0
    sortedTimestamps := make([]time.Time, 0, len(data))
    for timestamp := range data {
        sortedTimestamps = append(sortedTimestamps, timestamp)
    }
    sort.Slice(sortedTimestamps, func(i, j int) bool {
        return sortedTimestamps[i].Before(sortedTimestamps[j])
    })

    for i, timestamp := range sortedTimestamps {
        if timestamp.Before(intervalBegin) {
            continue
        }
        if timestamp.After(intervalEnd) {
            break
        }
        nextTimestamp := intervalEnd
        if i+1 < len(sortedTimestamps) {
            nextTimestamp = sortedTimestamps[i+1]
        }
        if nextTimestamp.After(intervalEnd) {
            nextTimestamp = intervalEnd
        }
        durationHours := nextTimestamp.Sub(timestamp).Hours()
        total += int(float64(data[timestamp]) * durationHours)
    }
    return total
}

// DayProduction returns the day production.
func (e *Estimate) DayProduction(specificDate time.Time) int {
    for date, production := range e.WhDays {
        if date.Equal(specificDate) {
            return production
        }
    }
    return 0
}

// Now returns the current timestamp in the API timezone.
func (e *Estimate) Now() time.Time {
    return time.Now().In(e.ApiTimezone)
}

// PeakProductionTime returns the peak time on a specific date.
func (e *Estimate) PeakProductionTime(specificDate time.Time) time.Time {
    var peakTime time.Time
    var maxWatt int
    for timestamp, watt := range e.Watts {
        if timestamp.Truncate(24 * time.Hour).Equal(specificDate) && watt > maxWatt {
            maxWatt = watt
            peakTime = timestamp
        }
    }
    if peakTime.IsZero() {
        panic("No peak production time found")
    }
    return peakTime
}

// PowerProductionAtTime returns the estimated power production at a specific time.
func (e *Estimate) PowerProductionAtTime(at time.Time) int {
    return TimedValue(at, e.Watts)
}

// SumEnergyProduction returns the sum of the energy production.
func (e *Estimate) SumEnergyProduction(periodHours int) int {
    now := e.Now().Truncate(time.Hour).Add(time.Hour - time.Nanosecond)
    until := now.Add(time.Duration(periodHours) * time.Hour)
    return IntervalValueSum(now, until, e.WhPeriod)
}