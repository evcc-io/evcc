package server

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	promActiveVehicles *prometheus.GaugeVec
)

func registerMetrics() {
	promActiveVehicles = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "soc",
		Name:      "active_vehicles",
	}, []string{"token", "brand"})

	if err := prometheus.Register(promActiveVehicles); err != nil {
		log.Fatal(err)
	}
}

func updateActiveVehiclesMetric(token, typ string, delta int) {
	g, err := promActiveVehicles.GetMetricWith(prometheus.Labels{
		"token": token,
		"brand": typ,
	})
	if err != nil {
		log.Println("get metrics:", err)
		return
	}

	g.Add(float64(delta))
}
