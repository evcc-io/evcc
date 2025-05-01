package plugchoice

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/util/request"
)

// FindUUIDByIdentity searches through all available chargers to find the UUID based on the identity
func FindUUIDByIdentity(client *request.Helper, baseURI string, identity string) (string, error) {
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

		// If no more data is returned, break the loop
		if len(res.Data) == 0 {
			break
		}
	}

	return "", errors.New("charger not found")
}
