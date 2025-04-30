package plugchoice

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// FindChargerUUIDByIdentity searches through all available chargers to find the UUID based on the identity
func FindChargerUUIDByIdentity(log *util.Logger, client *request.Helper, baseURI string, identity string) (string, error) {
	baseURI = baseURI + "/api/v3/chargers"

	// Start with the first page and keep going until we find it or run out of pages
	page := 1
	maxPages := 300 // Safety cap to prevent infinite loops

	log.TRACE.Printf("searching for charger with identity %s", identity)

	for page <= maxPages {
		uri := fmt.Sprintf("%s?page=%d", baseURI, page)
		log.TRACE.Printf("fetching chargers page %d", page)

		var res ChargerListResponse
		if err := client.GetJSON(uri, &res); err != nil {
			return "", fmt.Errorf("error fetching chargers list: %w", err)
		}

		// If no chargers on this page, we're done
		if len(res.Data) == 0 {
			log.TRACE.Printf("no chargers found on page %d, stopping search", page)
			break
		}

		// Search for the identity in this page
		for _, charger := range res.Data {
			if charger.Identity == identity {
				log.TRACE.Printf("found charger with identity %s: uuid=%s on page %d", identity, charger.UUID, page)
				return charger.UUID, nil
			}
		}

		// Move to the next page
		page++

		// If we're at the last page according to the API, no need to continue
		if res.Meta.LastPage > 0 && page > res.Meta.LastPage {
			log.TRACE.Printf("reached last page (%d), stopping search", res.Meta.LastPage)
			break
		}
	}

	if page > maxPages {
		log.TRACE.Printf("reached maximum number of pages (%d), stopping search", maxPages)
	}

	return "", fmt.Errorf("no charger found with identity %s after checking %d pages", identity, page-1)
}

// GetNextPageURL parses the next page URL from the response
func GetNextPageURL(next string) (string, error) {
	if next == "" {
		return "", nil
	}

	u, err := url.Parse(next)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
