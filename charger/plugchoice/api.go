package plugchoice

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// FindChargerUUIDByIdentity searches through all available chargers to find the UUID based on the identity
func FindChargerUUIDByIdentity(log *util.Logger, client *request.Helper, baseURI string, identity string) (string, error) {
	baseURI = baseURI + "/api/v3/chargers"

	for page := 1; page < 10; page++ {
		uri := fmt.Sprintf("%s?page=%d", baseURI, page)

		var res ChargerListResponse
		if err := client.GetJSON(uri, &res); err != nil {
			return "", fmt.Errorf("fetching chargers: %w", err)
		}

		// Search for the identity in this page
		for _, charger := range res.Data {
			if charger.Identity == identity {
				return charger.UUID, nil
			}
		}

		// If we're at the last page according to the API, no need to continue
		if res.Meta.LastPage > 0 && page >= res.Meta.LastPage {
			break
		}
	}

	return "", errors.New("charger not found")
}
