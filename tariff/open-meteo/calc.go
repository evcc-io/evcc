package openmeteo

import (
    "fmt"
    "time"
    "math"
    "github.com/evcc-io/evcc/api"
    "github.com/evcc-io/evcc/util/shortrfc3339"

)

const (
    G_NOCT           = 800.0
    G_STC            = 1000.0
    TEMP_STC_CELL    = 25.0
    ALPHA_TEMP       = -0.0045
    ROSS_COEFFICIENT = 0.0342
)


// The following functions are used to generate power estimates based on weather data,
// calculate damping coefficients for power generation, and process sunrise and sunset times
// to adjust power generation estimates accordingly.

// genPower generates power based on given parameters.
func (t *OpenMeteo) GenPower(gti, tAmb, eff, dcWp float64) float64 {
    tempCell := tAmb + (gti/G_NOCT)*ROSS_COEFFICIENT
    power := dcWp * (gti / G_STC) * (1 + ALPHA_TEMP*(tempCell-TEMP_STC_CELL)) * eff
    // DEBUG
    fmt.Println("DEBUG: GenPower - gti:", gti, "tAmb:", tAmb, "eff:", eff, "dcWp:", dcWp, "Power:", power)
    t.log.INFO.Printf("DEBUG: GenPower - gti: %f, tAmb: %f, eff: %f, dcWp: %f, Power: %f", gti, tAmb, eff, dcWp, power)
    // DEBUG
    return max(0, power)
}

// calculate calculates the power generation estimates.s
func (t *OpenMeteo) Calculate(res Response) error {
    var utcOffset *int
    whDays := make(map[time.Time]float64)

    for i := range t.Latitude {
        if utcOffset == nil {
            utcOffset = &res.UtcOffsetSeconds
        } else if *utcOffset != res.UtcOffsetSeconds {
            return fmt.Errorf("the UTC offset is not the same for all locations")
        }

        timeArr := make([]time.Time, len(res.Hourly.Time))
        berlinLocation, err := time.LoadLocation("Europe/Berlin")
        if err != nil {
            return err
        }

        for j, timeStr := range res.Hourly.Time {
            formattedTime, err := parseISO8601ToRFC3339(timeStr, res.Location)
            if err != nil {
                return err
            }

            parsedTime, err := time.Parse(time.RFC3339, formattedTime)
            if err != nil {
                return err
            }

            timeArr[j] = parsedTime.In(berlinLocation)

            fmt.Println("DEBUG: Parsed timestamp (Berlin):", timeArr[j])
            t.log.INFO.Printf("DEBUG: Parsed timestamp (Berlin) index=%d, value=%s", j, timeArr[j].Format(time.RFC3339))
        }

        sunriseDict, sunsetDict, err := t.CalculateSunriseSunset(&res)
        if err != nil {
            return err
        }

        dampingFactors := t.CalculateDampingFactors(timeArr, sunriseDict, sunsetDict, t.DampingMorning[i], t.DampingEvening[i])

        wAvg, wInst := t.CalculatePower(timeArr, &res, i, dampingFactors)
        t.ClampPower(wAvg, wInst)

        whPeriod, whPeriodCount := t.CalculateWhPeriod(wAvg)

        for tTime, power := range wAvg {
            hour := tTime.Truncate(time.Hour)
            whPeriod[hour] += power
            whPeriodCount[hour]++
        }

        for tTime, power := range whPeriod {
            day := tTime.Truncate(24 * time.Hour)
            whDays[day] += power
        }

        t.log.INFO.Printf("rates: %+v", whPeriod)

// Konvertiere whPeriod (map) in eine api.Rates Slice
rates := make(api.Rates, 0, len(whPeriod))

for start, price := range whPeriod {
    rate := api.Rate{
        Start: start.UTC(),
        End:   start.Add(time.Hour).UTC(),
        Price: math.Round(price),
    }

    // Debug-Ausgabe für jede Rate
    //fmt.Printf("%s %s %.0f\n",
    //    rate.Start.Format("2006-01-02 15:04:05 -0700 UTC"),
    //    rate.End.Format("2006-01-02 15:04:05 -0700 UTC"),
    //    rate.Price)

    rates = append(rates, rate)
    }

    // Speichern der Daten im gleichen Format wie Solcast
    t.Data.Set(rates)
    
    // Debugging-Ausgabe für die gesamte Struktur
    t.log.INFO.Printf("t.Data: %+v", rates)

    }
    return nil
}



// calculateDampingCoefficient calculates the damping coefficient for a given time.
func (t *OpenMeteo) CalculateDampingCoefficient(tTime, sunrise, sunset time.Time, dampingMorning, dampingEvening float64) float64 {
    morningStart := sunrise
    morningEnd := sunrise.Add(sunset.Sub(sunrise) / 2)
    eveningStart := morningEnd
    eveningEnd := sunset

    linearDamping := func(start, end time.Time, damping float64) float64 {
        duration := end.Sub(start)
        elapsed := tTime.Sub(start)
        damping = 1.0 - damping
        return (elapsed.Seconds() / duration.Seconds()) * (1.0 - damping) + damping
    }

    if morningStart.Before(tTime) && tTime.Before(morningEnd) {
        return linearDamping(morningStart, morningEnd, dampingMorning)
    }

    if eveningStart.Before(tTime) && tTime.Before(eveningEnd) {
        return linearDamping(eveningEnd, eveningStart, dampingEvening)
    }

    return 1
}

// calculateSunriseSunset calculates sunrise and sunset times from the response.
func (t *OpenMeteo) CalculateSunriseSunset(res *Response) (map[time.Time]time.Time, map[time.Time]time.Time, error) {
    sunriseDict := make(map[time.Time]time.Time)
    sunsetDict := make(map[time.Time]time.Time)

// Lade die Zeitzone dynamisch
berlinLocation, err := time.LoadLocation(res.Location)
if err != nil {
    return nil, nil, fmt.Errorf("invalid timezone received: %s", res.Location)
}

for i, timeStr := range res.Daily.Sunrise {
    var tssunrise shortrfc3339.Timestamp
    err := tssunrise.UnmarshalJSON([]byte(`"` + timeStr + `Z"`))
    if err != nil {
        return nil, nil, err
    }

    var tssunset shortrfc3339.Timestamp
    err = tssunset.UnmarshalJSON([]byte(`"` + res.Daily.Sunset[i] + `Z"`))
    if err != nil {
        return nil, nil, err
    }

    // Konvertiere Zeiten in dynamisch geladene Zeitzone
    sunriseDict[tssunrise.Time.UTC().Truncate(24*time.Hour)] = tssunrise.Time.In(berlinLocation)
    sunsetDict[tssunset.Time.UTC().Truncate(24*time.Hour)] = tssunset.Time.In(berlinLocation)
}


    return sunriseDict, sunsetDict, nil
}

// calculateDampingFactors calculates damping factors for a given set of time intervals.
func (t *OpenMeteo) CalculateDampingFactors(timeArr []time.Time, sunriseDict, sunsetDict map[time.Time]time.Time, dampingMorning, dampingEvening float64) []float64 {
    dampingFactors := make([]float64, len(timeArr))
    for i, tTime := range timeArr {
        dayKey := tTime.Truncate(24 * time.Hour)
        sunrise, ok1 := sunriseDict[dayKey]
        sunset, ok2 := sunsetDict[dayKey]

        if !ok1 || !ok2 {
            dampingFactors[i] = 1
            continue
        }

        dampingFactors[i] = t.CalculateDampingCoefficient(tTime, sunrise, sunset, dampingMorning, dampingEvening)
    }
    return dampingFactors
}

// CalculatePower calculates the average and instantaneous power generation.
func (t *OpenMeteo) CalculatePower(timeArr []time.Time, res *Response, i int, dampingFactors []float64) (map[time.Time]float64, map[time.Time]float64) {
    wAvg := make(map[time.Time]float64)
    wInst := make(map[time.Time]float64)
    dcWp := t.DcKwp[i] * 1000

    for j, tTime := range timeArr {
        if j-1 < 0 || j >= len(res.Hourly.GlobalTiltedIrradiance) || j >= len(res.Hourly.Temperature2m) {
            continue
        }

        if res.Hourly.GlobalTiltedIrradiance[j] == 0 || res.Hourly.Temperature2m[j] == 0 {
            continue
        }

        gAvg := res.Hourly.GlobalTiltedIrradiance[j]
        gInst := res.Hourly.GlobalTiltedIrradiance[j]
        tempAvg := (res.Hourly.Temperature2m[j] + res.Hourly.Temperature2m[j-1]) / 2
        tempInst := res.Hourly.Temperature2m[j-1]
        timeStart := tTime.Add(time.Hour) //tTime.Truncate(time.Hour) //tTime.Add(-15 * time.Minute)
        effDamped := t.EfficiencyFactor[i] * dampingFactors[j]

        wAvg[timeStart] += t.GenPower(gAvg, tempAvg, effDamped, dcWp)
        genPowerInst := t.GenPower(gInst, tempInst, effDamped, dcWp)

        fmt.Println("DEBUG: Zeit in wInst gespeichert:", timeStart, " -> ", genPowerInst)

        fmt.Println("DEBUG: wInst Calculation - Time:", timeStart, "gInst:", gInst, "tempInst:", tempInst, "effDamped:", effDamped, "dcWp:", dcWp, "Power:", genPowerInst)
        t.log.INFO.Printf("DEBUG: wInst Calculation - Time: %s, gInst: %f, tempInst: %f, effDamped: %f, dcWp: %f, Power: %f",
        timeStart, gInst, tempInst, effDamped, dcWp, genPowerInst)

        if _, exists := wInst[timeStart]; !exists {
            wInst[timeStart] = 0
        }
        wInst[timeStart] += genPowerInst // DEBUG ORIGINAL wInst[timeStart] += genPowerInst

    }

    return wAvg, wInst
}

// clampPower clamps the power values to the maximum AC power.
//func (t *OpenMeteo) ClampPower(wAvg, wInst map[time.Time]float64) {
//    acWp := t.AcKwp * 1000
//    for tTime := range wAvg {
//        wAvg[tTime] = min(wAvg[tTime], acWp)
//    }
//    for tTime := range wInst {
//        wInst[tTime] = min(wInst[tTime], acWp)
//    }
//}
// ClampPower clamps the power values to the maximum AC power and logs the results.
//func (t *OpenMeteo) ClampPower(wAvg, wInst map[time.Time]float64) {
//    acWp := t.AcKwp * 1000
//    for tTime := range wAvg {
//        original := wAvg[tTime]
//        wAvg[tTime] = min(wAvg[tTime], acWp)
//        t.log.INFO.Printf("DEBUG: ClampPower - Time: %s, Original wAvg: %f, Clamped wAvg: %f", 
//            tTime.Format(time.RFC3339), original, wAvg[tTime])
//    }
//    for tTime := range wInst {
//        original := wInst[tTime]
//        wInst[tTime] = min(wInst[tTime], acWp)
//        t.log.INFO.Printf("DEBUG: ClampPower - Time: %s, Original wInst: %f, Clamped wInst: %f", 
//            tTime.Format(time.RFC3339), original, wInst[tTime])
//    }
//}
//func (t *OpenMeteo) ClampPower(wAvg, wInst map[time.Time]float64) {
//    acWp := t.AcKwp * 1000
//    for tTime := range wAvg {
//        wAvg[tTime] = min(wAvg[tTime], acWp)
//    }
//    for tTime := range wInst {
//        wInst[tTime] = min(wInst[tTime], acWp)
//    }
//}

// ClampPower begrenzt die Leistungswerte auf die maximale AC-Leistung
func (t *OpenMeteo) ClampPower(wAvg, wInst map[time.Time]float64) {
    acWp := t.AcKwp * 1000

    for tTime := range wAvg {
        wAvg[tTime] = math.Min(wAvg[tTime], acWp)
    }
    for tTime := range wInst {
        wInst[tTime] = math.Min(wInst[tTime], acWp)
    }
}

// CalculateWhPeriod calculates the Wh period from average power values.
func (t *OpenMeteo) CalculateWhPeriod(wAvg map[time.Time]float64) (map[time.Time]float64, map[time.Time]int) {
    whPeriod := make(map[time.Time]float64)
    whPeriodCount := make(map[time.Time]int)
    for tTime, power := range wAvg {
        hour := tTime.Truncate(time.Hour)
        whPeriod[hour] += power
        whPeriodCount[hour]++
    }
    for tTime := range whPeriod {
        whPeriod[tTime] /= float64(whPeriodCount[tTime])
    }
    return whPeriod, whPeriodCount
}

// CalculateWhDays calculates the Wh days from Wh period values.
func (t *OpenMeteo) CalculateWhDays(whPeriod map[time.Time]float64) map[time.Time]float64 {
    whDays := make(map[time.Time]float64)
    for tTime, power := range whPeriod {
        day := tTime.Truncate(24 * time.Hour)
        whDays[day] += power
    }
return whDays
}

// max returns the maximum of two float64 values.
func max(a, b float64) float64 {
if a > b {
return a
    }
    return b
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}

// Umrechnung der Zeitwerte von ISO 8601 ohne Z, in RFC 3339
func parseISO8601ToRFC3339(timeStr, timezone string) (string, error) {
    loc, err := time.LoadLocation(timezone) // Lade die passende Zeitzone
    if err != nil {
        return "", fmt.Errorf("invalid timezone: %s", err)
    }

    parsedTime, err := time.ParseInLocation("2006-01-02T15:04", timeStr, loc)
    if err != nil {
        return "", fmt.Errorf("invalid time format: %s", err)
    }

    return parsedTime.Format(time.RFC3339), nil
}
