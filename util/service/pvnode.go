package service

import (
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sites", getPvnodeSites)

	service.Register("pvnode", mux)
}

const pvnodeSitesURI = "https://api.pvnode.com/v2/sites/"

type pvnodeSite struct {
	ID string
}

// getPvnodeSites lists the account's site ids, driving site selection in the
// template. A site id given in the request is looked up individually and added
// if the listing does not contain it. The config UI polls while the form is
// filled, hence a missing or rejected api key yields an empty list.
func getPvnodeSites(w http.ResponseWriter, req *http.Request) {
	res := []string{}
	defer func() { jsonWrite(w, res) }()

	q := req.URL.Query()

	apikey := q.Get("apikey")
	if apikey == "" {
		return
	}

	client := request.NewHelper(util.NewLogger("pvnode").Redact(apikey))
	client.Transport = transport.BearerAuth(apikey, client.Transport)

	var sites []pvnodeSite
	if err := client.GetJSON(pvnodeSitesURI, &sites); err == nil {
		for _, site := range sites {
			res = append(res, site.ID)
		}
	}

	if id := q.Get("site_id"); id != "" && !slices.Contains(res, id) {
		var site pvnodeSite
		if err := client.GetJSON(pvnodeSitesURI+id, &site); err == nil {
			res = append(res, site.ID)
		}
	}

	slices.Sort(res)
}
