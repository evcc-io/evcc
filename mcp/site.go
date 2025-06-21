package mcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/mark3labs/mcp-go/mcp"
)

func loadpointsHandler(site site.API) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		count := len(site.Loadpoints())
		return mcp.NewToolResultText(strconv.Itoa(count)), nil
	}
}

func solarForecastHandler(site site.API) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		solar := site.GetTariff(api.TariffUsageSolar)
		if solar == nil {
			return nil, errors.New("solar tariff not found")
		}

		rates := tariff.Forecast(solar)
		if len(rates) == 0 {
			return nil, errors.New("solar forecast data not available")
		}

		// TODO use global function
		var total float64
		for _, rate := range rates {
			if rate.Start.After(time.Now()) && !rate.End.After(now.EndOfDay()) {
				total += rate.Value * rate.End.Sub(rate.Start).Hours() // Wh
			}
		}

		return mcp.NewToolResultText(fmt.Sprintf("%.2f kWH", total/1e3)), nil
	}
}
