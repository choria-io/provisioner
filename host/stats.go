// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package host

import "github.com/prometheus/client_golang/prometheus"

var (
	rpcDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "choria_provisioner_rpc_time",
		Help: "How long it took to perform RPC requests",
	}, []string{"site", "rpc"})

	helperDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "choria_provisioner_helper_time",
		Help: "How long it took to run the helper",
	}, []string{"site"})

	rpcErrCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_rpc_errors",
		Help: "How many rpc related errors were encountered",
	}, []string{"site", "rpc"})

	helperErrCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_provisioner_helper_errors",
		Help: "How many helper related errors were encountered",
	}, []string{"site"})
)

func init() {
	prometheus.MustRegister(rpcDuration)
	prometheus.MustRegister(helperDuration)
	prometheus.MustRegister(rpcErrCtr)
	prometheus.MustRegister(helperErrCtr)
}
