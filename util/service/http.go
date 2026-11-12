package service

import (
	"cmp"
	"context"
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

// parseQuery compiles a jq expression, an empty expression yields nil
func parseQuery(expr string) (*gojq.Query, error) {
	if expr == "" {
		return nil, nil
	}
	return gojq.Parse(expr)
}

// first returns the first result of query applied to v, or nil. The request
// context bounds the evaluation like the jq query itself.
func first(ctx context.Context, query *gojq.Query, v any) any {
	if query == nil {
		return nil
	}
	res, ok := query.RunWithContext(ctx, v).Next()
	if !ok {
		return nil
	}
	if _, ok := res.(error); ok {
		return nil
	}
	return res
}

// httpGet reads uri and maps the jq result elements to options via the optional
// value, label and hint expressions. The config UI polls, hence errors yield an empty list.
func httpGet(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()

	uri := q.Get("uri")
	if uri == "" {
		jsonError(w, http.StatusBadRequest, errors.New("missing uri"))
		return
	}

	var queries [4]*gojq.Query
	for i, expr := range []string{
		cmp.Or(q.Get("jq"), "."),
		cmp.Or(q.Get("value"), "."),
		q.Get("label"),
		q.Get("hint"),
	} {
		var err error
		if queries[i], err = parseQuery(expr); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
	}
	query, value, label, hint := queries[0], queries[1], queries[2], queries[3]

	log := util.NewLogger("http")
	client := request.NewHelper(log)

	if token := q.Get("token"); token != "" {
		log.Redact(token)
		client.Transport = transport.BearerAuth(token, client.Transport)
	}

	res := []service.Option{}
	defer func() { jsonWrite(w, res) }()

	body, err := client.GetBody(uri)
	if err != nil {
		return
	}

	val, err := jq.QueryContext(req.Context(), query, body)
	if err != nil {
		return
	}

	slice, ok := val.([]any)
	if !ok {
		slice = []any{val}
	}

	for _, v := range slice {
		val := first(req.Context(), value, v)
		if val == nil {
			continue
		}

		// match keeps the single-value auto-fill
		res = append(res, service.Option{
			Value: cast.ToString(val),
			Label: cast.ToString(first(req.Context(), label, v)),
			Hint:  cast.ToString(first(req.Context(), hint, v)),
			Match: true,
		})
	}
}
