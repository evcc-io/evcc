package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/mark3labs/mcp-go/mcp"
)

func siteAllLoadpointsAsRessourcesHandler(site site.API) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var res []mcp.Content

		for idx, lp := range site.Loadpoints() {
			uri := fmt.Sprintf("loadpoint://%d", idx)
			res = append(res, mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "application/json",
					Text:     lp.GetTitle(),
				},
			})
		}

		return &mcp.CallToolResult{
			Content: res,
		}, nil
	}
}

func siteAllLoadpointsHandler(site site.API) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var res []Loadpoint
		for _, lp := range site.Loadpoints() {
			res = append(res, loadpointDetails(lp))
		}

		b, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(b)), nil
	}
}

// func siteLoadpointsHandler(site site.API) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
// 		id := req.GetInt("id", -1)
// 		if id == 0 {
// 			if name := req.GetString("name", ""); name != "" {
// 				_, id, _ = lo.FindIndexOf(site.Loadpoints(), func(lp loadpoint.API) bool {
// 					return strings.EqualFold(lp.GetTitle(), name)
// 				})
// 			}
// 		}

// 		if id >= len(site.Loadpoints()) {
// 			return mcp.NewToolResultErrorf("invalid loadpoint id: %d", id), nil
// 		}

// 		if id >= 0 {
// 			b, _ := json.Marshal(loadpointDetails(site.Loadpoints()[id]))
// 			return mcp.NewToolResultText(string(b)), nil
// 		}

// 		var res []Loadpoint
// 		for _, lp := range site.Loadpoints() {
// 			res = append(res, loadpointDetails(lp))
// 		}

// 		b, _ := json.Marshal(res)
// 		return mcp.NewToolResultText(string(b)), nil
// 	}
// }

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
