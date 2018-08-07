package host

import "github.com/prometheus/client_golang/prometheus"

var (
	rpcDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "choria_provisioner_rpc_time",
		Help: "How long it took to perform RPC requests",
	}, []string{"site", "rpc"})

	rpcErrCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_rpc_errors",
		Help: "How many rpc related errors were encountered",
	}, []string{"site", "rpc"})
)

func init() {
	prometheus.MustRegister(rpcDuration)
	prometheus.MustRegister(rpcErrCtr)
}
