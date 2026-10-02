package eudataact

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
)

// DatasetSummary describes a downloaded and parsed EU Data Act dataset.
type DatasetSummary struct {
	Name       string
	CreatedOn  time.Time
	PointCount int
}

// LatestDataset downloads and parses the newest content dataset for vin.
func (v *API) LatestDataset(vin string) (DatasetSummary, error) {
	identifier, err := v.identifier(vin)
	if err != nil {
		return DatasetSummary{}, fmt.Errorf("dataset metadata: %w", err)
	}

	datasets, err := v.datasets(vin, identifier)
	if err != nil {
		return DatasetSummary{}, fmt.Errorf("list datasets: %w", err)
	}
	datasets = contentDatasets(datasets)
	if len(datasets) == 0 {
		return DatasetSummary{}, api.ErrNotAvailable
	}

	newest := datasets[len(datasets)-1]
	archive, err := v.download(vin, identifier, newest.Name)
	if err != nil {
		return DatasetSummary{}, fmt.Errorf("download dataset %s: %w", newest.Name, err)
	}

	points, err := parseDataset(archive)
	if err != nil {
		return DatasetSummary{}, fmt.Errorf("parse dataset %s: %w", newest.Name, err)
	}

	return DatasetSummary{
		Name:       newest.Name,
		CreatedOn:  newest.CreatedOn,
		PointCount: len(points),
	}, nil
}
