// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

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
