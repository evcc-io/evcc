package viessmann

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
)

var serviceMux = http.NewServeMux()

func init() {
	serviceMux.HandleFunc("GET /equipment", getEquipment)

	service.Register("viessmann", serviceMux)
}

// errUnauthorized indicates that the user has not (yet) authorized. The config
// UI polls the service while the form is being filled, hence this is expected.
var errUnauthorized = errors.New("unauthorized")

// equipment reuses the OAuth instance created for the same client, so results
// appear once the user has authorized via the UI.
func equipment(req *http.Request) ([]Installation, error) {
	q := req.URL.Query()
	clientID, redirectURI := q.Get("clientid"), q.Get("redirecturi")

	if clientID == "" {
		return nil, errUnauthorized
	}

	log := util.NewLogger("viessmann").Redact(clientID)
	ctx := util.WithLogger(context.Background(), log)

	ts, err := NewOAuth(ctx, clientID, redirectURI, q.Get("gateway_serial"))
	if err != nil {
		return nil, err
	}

	// no values until the user has authorized
	if _, err := ts.Token(); err != nil {
		return nil, errUnauthorized
	}

	return NewAPI(log, ApiURI, ts).Installations()
}

// values extracts the installation ids or, with gateways set, the deduplicated
// gateway serials of the given installations.
func values(installations []Installation, gateways bool) []string {
	res := []string{}

	for _, inst := range installations {
		if !gateways {
			res = append(res, strconv.Itoa(inst.ID))
			continue
		}

		for _, gw := range inst.Gateways {
			if !slices.Contains(res, gw.Serial) {
				res = append(res, gw.Serial)
			}
		}
	}

	slices.Sort(res)

	return res
}

// getEquipment lists the account's installation ids or, with detail=gateways,
// their gateway serials, driving the respective selection in the template.
func getEquipment(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	res := []string{}
	defer func() { _ = json.NewEncoder(w).Encode(res) }()

	installations, err := equipment(req)
	if err != nil {
		// unexpected errors are worth logging, missing authorization is not
		if !errors.Is(err, errUnauthorized) {
			util.NewLogger("viessmann").ERROR.Println(err)
		}
		return
	}

	res = values(installations, req.URL.Query().Get("detail") == "gateways")
}
