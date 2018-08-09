package hosts

import "github.com/prometheus/client_golang/prometheus"

var (
	discoveredCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_discovered",
		Help: "How many nodes were found through discovery",
	}, []string{"site"})

	eventsCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_event_discovered",
		Help: "How many nodes were found through receiving an event about them",
	}, []string{"site"})

	discoverCycleCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_discover_cycles",
		Help: "How many discovery cycles were ran",
	}, []string{"site"})

	errCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_discovery_errors",
		Help: "How many discovery related errors were encountered",
	}, []string{"site"})

	provErrCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_provision_errors",
		Help: "How many provision related errors were encountered",
	}, []string{"site"})

	busyWorkerGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_provisioner_busy_workers",
		Help: "How many workers are busy provisioning nodes",
	}, []string{"site"})
)

func init() {
	prometheus.MustRegister(discoveredCtr)
	prometheus.MustRegister(eventsCtr)
	prometheus.MustRegister(discoverCycleCtr)
	prometheus.MustRegister(errCtr)
	prometheus.MustRegister(provErrCtr)
	prometheus.MustRegister(busyWorkerGauge)
}
