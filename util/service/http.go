package service

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/itchyny/gojq"
	"github.com/spf13/cast"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /get", httpGet)

	service.Register("http", mux)
}

// httpGet reads uri and extracts the choices using the jq query, driving
// selection of remote resources in templates. The config UI polls while the
// form is filled, hence a rejected request yields an empty list.
func httpGet(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()

	uri := q.Get("uri")
	if uri == "" {
		jsonError(w, http.StatusBadRequest, errors.New("missing uri"))
		return
	}

	expr := q.Get("jq")
	if expr == "" {
		expr = "."
	}

	query, err := gojq.Parse(expr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	log := util.NewLogger("http")
	client := request.NewHelper(log)

	if token := q.Get("token"); token != "" {
		log.Redact(token)
		client.Transport = transport.BearerAuth(token, client.Transport)
	}

	res := []string{}
	defer func() { jsonWrite(w, res) }()

	body, err := client.GetBody(uri)
	if err != nil {
		return
	}

	val, err := jq.Query(query, body)
	if err != nil {
		return
	}

	if slice, ok := val.([]any); ok {
		for _, v := range slice {
			res = append(res, cast.ToString(v))
		}
		return
	}

	res = append(res, cast.ToString(val))
}
