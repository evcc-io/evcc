package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

type Home struct {
	ID                string
	TimeZone          string
	Address           Address
	MeteringPointData struct {
		GridCompany string
	}
}

type Address struct {
	Address1, PostalCode, City, Country string
}

type Subscription struct {
	ID        string
	Status    string
	PriceInfo struct {
		Current PriceInfo
		Today   []PriceInfo
		// Tomorrow []PriceInfo
	}
}

type PriceInfo struct {
	Level    string
	StartsAt time.Time
	Total    float64
	// Energy, Tax float64
}

type homes struct {
	Viewer struct {
		Homes []Home
	}
}

type prices struct {
	Viewer struct {
		Home struct {
			ID                  string
			TimeZone            string
			CurrentSubscription Subscription
		} `graphql:"home(id: $id)"`
	}
}

const homeID = "c70dcbe5-4485-4821-933d-a8a86452737b"

const awattarURI = "https://api.awattar.de/v1/marketdata"

type AwattarPrices struct {
	Data []PriceInfoA
}

type PriceInfoA struct {
	StartTimestamp time.Time `json:"start_timestamp"`
	EndTimestamp   time.Time `json:"end_timestamp"`
	Marketprice    float64   `json:"marketprice"`
	Unit           string    `json:"unit"`
}

func (p *PriceInfoA) UnmarshalJSON(data []byte) error {
	var s struct {
		StartTimestamp int64   `json:"start_timestamp"`
		EndTimestamp   int64   `json:"end_timestamp"`
		Marketprice    float64 `json:"marketprice"`
		Unit           string  `json:"unit"`
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		p.StartTimestamp = time.Unix(s.StartTimestamp/1e3, 0)
		p.EndTimestamp = time.Unix(s.EndTimestamp/1e3, 0)
		p.Marketprice = s.Marketprice
		p.Unit = s.Unit
	}

	return err
}

func main() {
	token := "d1007ead2dc84a2b82f0de19451c5fb22112f7ae11d19bf2bedb224a003ff74a"

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	client := graphql.NewClient("https://api.tibber.com/v1-beta/gql", tc)

	qh := homes{}
	if err := client.Query(context.Background(), &qh, nil); err != nil {
		panic(err)
	}

	q := prices{}
	v := map[string]interface{}{
		"id": graphql.ID(qh.Viewer.Homes[0].ID),
	}

	if err := client.Query(context.Background(), &q, v); err != nil {
		panic(err)
	}

	fmt.Println(q)

	var res AwattarPrices
	h := request.NewHelper(util.NewLogger("foo"))
	if err := h.GetJSON(awattarURI, &res); err != nil {
		panic(err)
	}
	fmt.Println(res)
}
