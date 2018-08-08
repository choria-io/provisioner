package config

import "github.com/prometheus/client_golang/prometheus"

var (
	pausedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_provisioner_paused",
		Help: "Indicates if the provisioner is paused",
	}, []string{"site"})
)

func init() {
	prometheus.MustRegister(pausedGauge)
}
