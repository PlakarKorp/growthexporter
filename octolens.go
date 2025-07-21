package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var labels = make(map[string]prometheus.Gauge)

var sources = []string{
	"linkedin",
	"reddit",
	"twitter",
	"bluesky",
	"youtube",
	"dev",
	"github",
	"hackernews",
	"stackoverflow",
	"newsletters",
	"podcasts",
}

var keywords = []string{
	"plakar",
	"ptar",
	"kloset",
	"kapsul",
}

func init() {
	for _, source := range sources {
		for _, keyword := range keywords {
			metricName := "octolens_" + source + "_" + keyword
			metric := prometheus.NewGauge(prometheus.GaugeOpts{
				Name: metricName,
			})
			prometheus.MustRegister(metric)
		}
	}
}

func octolensHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("You've hit /octolens!"))

	// json
	// json['source'] = 'twitter'
	// json['source'] = 'hackernews'

	//

}
