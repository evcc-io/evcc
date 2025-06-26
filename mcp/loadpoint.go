package mcp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/mark3labs/mcp-go/mcp"
)

func loadpointStatusHandler(log *util.Logger, site site.API) func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		lpid := extractNumericID(req.Params.URI) // "users://123" -> "123"

		if lpid == 0 || lpid > len(site.Loadpoints()) {
			return nil, errors.New("invalid loadpoint ID")
		}

		lp := site.Loadpoints()[lpid-1] // Loadpoint IDs are 1-based
		status := lp.GetStatus()

		data := struct {
			Title     string `json:"title"`
			Connected bool   `json:"connected,omitempty"`
			Charging  bool   `json:"charging,omitempty"`
		}{
			Title:     lp.GetTitle(),
			Connected: status == api.StatusB || status == api.StatusC,
			Charging:  status == api.StatusC,
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	}
}
