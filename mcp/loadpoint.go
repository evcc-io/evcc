package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/mark3labs/mcp-go/mcp"
)

type Loadpoint struct {
	Title     string `json:"title"`
	Connected bool   `json:"connected,omitempty"`
	Charging  bool   `json:"charging,omitempty"`
}

func loadpointDetails(lp loadpoint.API) Loadpoint {
	status := lp.GetStatus()

	return Loadpoint{
		Title:     lp.GetTitle(),
		Connected: status == api.StatusB || status == api.StatusC,
		Charging:  status == api.StatusC,
	}
}

func allLoadpointsHandler(site site.API) func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		var res []mcp.ResourceContents

		for idx, lp := range site.Loadpoints() {
			uri := fmt.Sprintf("loadpoint://%d", idx)
			res = append(res, mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "application/json",
				Text:     lp.GetTitle(),
			})
		}

		return res, nil
	}
}

func loadpointStatusHandler(site site.API) func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := extractNumericID(req.Params.URI)
		if id < 0 || id >= len(site.Loadpoints()) {
			return nil, errors.New("invalid loadpoint id")
		}

		jr, err := NewJsonResourceContents(req.Params.URI, loadpointDetails(site.Loadpoints()[id]))
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			jr,
		}, nil
	}
}
