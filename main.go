package main

import (
	"encoding/csv"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "locust"
)

// Exporter structure
type Exporter struct {
	locustUp prometheus.Gauge

	locustCurrentResponseTimePercentileNinetyFifth,
	locustCurrentResponseTimePercentileFiftieth,
	locustNumRequests,
	locustNumFailures,
	locustAvgResponseTime,
	locustCurrentFailPerSec,
	locustSlavesDetail,
	locustMinResponseTime,
	locustMaxResponseTime,
	locustCurrentRps,
	locustMedianResponseTime,
	locustAvgContentLength,
	locustErrors *prometheus.GaugeVec
	totalScrapes prometheus.Counter
}

// NewExporter function
func NewExporter() (*Exporter, error) {

	return &Exporter{
		locustUp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "up",
				Help:      "The current health status of the server (1 = UP, 0 = DOWN).",
			},
		),
		locustCurrentResponseTimePercentileNinetyFifth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "current_response_time_percentile_95",
			},
			[]string{"method", "name"},
		),
		locustCurrentResponseTimePercentileFiftieth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "current_response_time_percentile_50",
			},
			[]string{"method", "name"},
		),
		locustSlavesDetail: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "slave",
				Name:      "detail",
				Help:      "The current status of a slave with user count",
			},
			[]string{"id", "state"},
		),
		locustNumRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "num_requests",
			},
			[]string{"method", "name"},
		),
		locustNumFailures: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "num_failures",
			},
			[]string{"method", "name"},
		),
		locustAvgResponseTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "avg_response_time",
			},
			[]string{"method", "name"},
		),
		locustCurrentFailPerSec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "current_fail_per_sec",
			},
			[]string{"method", "name"},
		),
		locustMinResponseTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "min_response_time",
			},
			[]string{"method", "name"},
		),
		locustMaxResponseTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "max_response_time",
			},
			[]string{"method", "name"},
		),
		locustCurrentRps: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "current_rps",
			},
			[]string{"method", "name"},
		),
		locustMedianResponseTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "median_response_time",
			},
			[]string{"method", "name"},
		),
		locustAvgContentLength: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "requests",
				Name:      "avg_content_length",
			},
			[]string{"method", "name"},
		),
		locustErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "errors",
				Help:      "The current number of errors.",
			},
			[]string{"method", "name", "error"},
		),
		totalScrapes: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "total_scrapes",
				Help:      "The total number of scrapes.",
			},
		),
	}, nil
}

// Describe function of Exporter
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {

	ch <- e.locustUp.Desc()

	e.locustCurrentResponseTimePercentileNinetyFifth.Describe(ch)
	e.locustCurrentResponseTimePercentileFiftieth.Describe(ch)

	e.locustNumRequests.Describe(ch)
	e.locustNumFailures.Describe(ch)
	e.locustAvgResponseTime.Describe(ch)
	e.locustCurrentFailPerSec.Describe(ch)
	e.locustMinResponseTime.Describe(ch)
	e.locustMaxResponseTime.Describe(ch)
	e.locustMedianResponseTime.Describe(ch)
	e.locustCurrentRps.Describe(ch)
	e.locustAvgContentLength.Describe(ch)
	e.locustErrors.Describe(ch)
	e.locustSlavesDetail.Describe(ch)
}

// Collect function of Exporter
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {

	up := e.scrape(ch)
	ch <- prometheus.MustNewConstMetric(e.locustUp.Desc(), prometheus.GaugeValue, up)
	e.locustNumRequests.Collect(ch)
	e.locustNumFailures.Collect(ch)
	e.locustAvgResponseTime.Collect(ch)
	e.locustCurrentFailPerSec.Collect(ch)
	e.locustMinResponseTime.Collect(ch)
	e.locustMaxResponseTime.Collect(ch)
	e.locustCurrentRps.Collect(ch)
	e.locustMedianResponseTime.Collect(ch)
	e.locustAvgContentLength.Collect(ch)
	e.locustErrors.Collect(ch)
	e.locustSlavesDetail.Collect(ch)
}

func getFloat64Element(record []string, i int) float64 {
	floatValue, _ := strconv.ParseFloat(record[i], 64)
	return floatValue
}
func (e *Exporter) scrape(ch chan<- prometheus.Metric) (up float64) {
	e.totalScrapes.Inc()

	csvfileStats, err := os.Open(*csvStatsFile)
	if err != nil {
		log.Errorln("Couldn't open the csv file", err)
		return 0
	}

	statsReader := csv.NewReader(csvfileStats)

	for {
		// Read each record from csv
		record, err := statsReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if record[0] != "Type" && record[0] != "None" {
			e.locustNumRequests.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 2))
			e.locustNumFailures.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 3))
			e.locustMedianResponseTime.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 4))
			e.locustAvgResponseTime.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 5))
			e.locustMinResponseTime.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 6))
			e.locustMaxResponseTime.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 7))
			e.locustAvgContentLength.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 8))
			e.locustCurrentRps.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 9))
			e.locustCurrentFailPerSec.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 10))
			e.locustCurrentResponseTimePercentileFiftieth.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 11))
			e.locustCurrentResponseTimePercentileNinetyFifth.WithLabelValues(record[0], record[1]).Set(getFloat64Element(record, 16))
		}
	}

	csvfileFailures, err := os.Open(*csvFailureFile)
	if err != nil {
		log.Errorln("Couldn't open the csv file", err)
		return 0
	}

	failuresReader := csv.NewReader(csvfileFailures)

	for {
		// Read each record from csv
		record, err := failuresReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if record[0] != "Method" {
			e.locustErrors.WithLabelValues(record[0], record[1], record[2]).Set(getFloat64Element(record, 3))
		}
	}

	return 1
}

var csvStatsFile *string
var csvFailureFile *string

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9646").Envar("LOCUST_EXPORTER_WEB_LISTEN_ADDRESS").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").Envar("LOCUST_EXPORTER_WEB_TELEMETRY_PATH").String()
	)

	csvStatsFile = kingpin.Flag("csv.stats", "Path under which will be locust csv file").Required().String()
	csvFailureFile = kingpin.Flag("csv.failures", "Path under which will be locust csv file").Required().String()

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("locust_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting locust_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter()
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("locustexporter"))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Locust Exporter</title></head><body><h1>Locust Exporter</h1><p><a href='` + *metricsPath + `'>Metrics</a></p></body></html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
