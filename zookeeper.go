package main

import (
	"bufio"
	"bytes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net"
	"net/http"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	// "net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type zookeeperCollector struct {
	upIndicator *prometheus.Desc
	metrics     map[string]zookeeperMetric
	sync.Mutex
	addr    string
	mux     *http.ServeMux
	options Options
}
type Options struct {
	Registry *prometheus.Registry
}
type zookeeperMetric struct {
	desc          *prometheus.Desc
	extract       func(string) float64
	extractLabels func(s []string) []string
	valType       prometheus.ValueType
}

func parseFloatOrZero(s string) float64 {
	res, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Warningf("Failed to parse to float64: %s", err)
		return 0.0
	}
	return res
}

// Exporter 的方法 ScrapeHandler 。 这个是主方法
func (c *zookeeperCollector) ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	// 获取target参数，IP:PORt
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}
	ipPort := strings.Split(target, ":")
	ip := net.ParseIP(ipPort[0])
	if ip == nil {
		http.Error(w, fmt.Sprintf("Invalid 'target' parameter, parse err: %s ", target), 400)
		return
	}
	c.addr = target
	log.Info(fmt.Sprintf("scrape target = %s", target))
	// 一个新的prometheus注册器
	registry := prometheus.NewRegistry()
	opts := c.options
	opts.Registry = registry

	_ = NewZookeeperCollector(target, &opts)
	promhttp.HandlerFor(
		registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},
	).ServeHTTP(w, r)
}

func NewZookeeperCollector(addr string, opts *Options) *zookeeperCollector {
	metricLabelsArray := []string{"target_host"}
	c := &zookeeperCollector{
		addr:        addr,
		options:     *opts,
		upIndicator: prometheus.NewDesc("zk_up", "Exporter successful", metricLabelsArray, nil),
		metrics: map[string]zookeeperMetric{
			"zk_avg_latency": {
				desc:    prometheus.NewDesc("zk_avg_latency", "Average latency of requests", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_max_latency": {
				desc:    prometheus.NewDesc("zk_max_latency", "Maximum seen latency of requests", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_min_latency": {
				desc:    prometheus.NewDesc("zk_min_latency", "Minimum seen latency of requests", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_packets_received": {
				desc:    prometheus.NewDesc("zk_packets_received", "Number of packets received", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.CounterValue,
			},
			"zk_packets_sent": {
				desc:    prometheus.NewDesc("zk_packets_sent", "Number of packets sent", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.CounterValue,
			},
			"zk_num_alive_connections": {
				desc:    prometheus.NewDesc("zk_num_alive_connections", "Number of active connections", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_outstanding_requests": {
				desc:    prometheus.NewDesc("zk_outstanding_requests", "Number of outstanding requests", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_server_state": {
				desc:    prometheus.NewDesc("zk_server_state", "Server state (leader/follower)", []string{"state", "target_host"}, nil),
				extract: func(s string) float64 { return 1 },
				extractLabels: func(s []string) []string {
					return s
				},
				valType: prometheus.UntypedValue,
			},
			"zk_znode_count": {
				desc:    prometheus.NewDesc("zk_znode_count", "Number of znodes", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_watch_count": {
				desc:    prometheus.NewDesc("zk_watch_count", "Number of watches", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_ephemerals_count": {
				desc:    prometheus.NewDesc("zk_ephemerals_count", "Number of ephemeral nodes", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_approximate_data_size": {
				desc:    prometheus.NewDesc("zk_approximate_data_size", "Approximate size of data set", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_open_file_descriptor_count": {
				desc:    prometheus.NewDesc("zk_open_file_descriptor_count", "Number of open file descriptors", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_max_file_descriptor_count": {
				desc:    prometheus.NewDesc("zk_max_file_descriptor_count", "Maximum number of open file descriptors", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.CounterValue,
			},
			"zk_followers": {
				desc:    prometheus.NewDesc("zk_followers", "Number of followers", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_synced_followers": {
				desc:    prometheus.NewDesc("zk_synced_followers", "Number of followers in sync", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_pending_syncs": {
				desc:    prometheus.NewDesc("zk_pending_syncs", "Number of followers with syncronizations pending", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_wchs_watch_connections": {
				desc:    prometheus.NewDesc("zk_wchs_watch_connections", "Number of  connections", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_wchs_total_watch": {
				desc:    prometheus.NewDesc("zk_wchs_total_watch", "Number of  watchs", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
			"zk_wchs_watch_paths": {
				desc:    prometheus.NewDesc("zk_wchs_watch_paths", "Number of  watchs paths", metricLabelsArray, nil),
				extract: func(s string) float64 { return parseFloatOrZero(s) },
				valType: prometheus.GaugeValue,
			},
		},
	}
	c.mux = http.NewServeMux()
	if c.options.Registry != nil {
		c.options.Registry.MustRegister(c)
	}
	c.mux.HandleFunc("/scrape", logPanics(c.ScrapeHandler))
	c.mux.HandleFunc("/", rootHandler)
	return c
}
func (c *zookeeperCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

// 传入zookeeprcollector
func (c *zookeeperCollector) Describe(ch chan<- *prometheus.Desc) {
	log.Debugf("Sending %d metrics descriptions", len(c.metrics))
	for _, i := range c.metrics {
		ch <- i.desc
	}
}

func (c *zookeeperCollector) Collect(ch chan<- prometheus.Metric) {
	log.Info("Fetching metrics from Zookeeper")

	data, ok := sendZkCommand(c.addr, "mntr")

	if !ok {
		log.Error("Failed to fetch metrics")
		ch <- prometheus.MustNewConstMetric(c.upIndicator, prometheus.GaugeValue, 0)
		return
	}
	data = strings.TrimSpace(data)
	data2, ok := sendZkCommand(c.addr, "wchs")
	data4 := strings.Replace(data2, "\n", "", -1)
	data5 := strings.Replace(data4, ":", " ", -1)
	data = data + "\n" +
		"zk_wchs_watch_connections" + "\t" + strings.SplitN(data5, " ", -1)[0]
	data = data + "\n" +
		"zk_wchs_total_watch" + "\t" + strings.SplitN(data5, " ", -1)[6]
	data = data + "\n" +
		"zk_wchs_watch_paths" + "\t" + strings.SplitN(data5, " ", -1)[3]
	// fmt.Printf("v2 type:%T\n", data)
	status := 1.0
	for _, line := range strings.Split(data, "\n") {
		parts := strings.Split(line, "\t")
		// fmt.Println(parts)
		if len(parts) != 2 {
			log.WithFields(log.Fields{"data": line}).Warn("Unexpected format of returned data, expected tab-separated key/value.")
			status = 0
			continue
		}
		label, value := parts[0], parts[1]
		metric, ok := c.metrics[label]
		if ok {
			log.Debug(fmt.Sprintf("Sending metric %s=%s", label, value))
			if metric.extractLabels != nil {
				ch <- prometheus.MustNewConstMetric(metric.desc, metric.valType, metric.extract(value), metric.extractLabels([]string{value, c.addr})...)
			} else {
				ch <- prometheus.MustNewConstMetric(metric.desc, metric.valType, metric.extract(value), []string{c.addr}...)
			}
		}
	}
	ch <- prometheus.MustNewConstMetric(c.upIndicator, prometheus.GaugeValue, status,[]string{c.addr}...)
	resetStatistics(c.addr)
}
func resetStatistics(addr string) {
	log.Info("Resetting Zookeeper statistics")
	_, ok := sendZkCommand(addr, "srst")
	if !ok {
		log.Warning("Failed to reset statistics")
	}
}

const (
	timeoutSeconds = 5
)

func sendZkCommand(addr string, fourLetterWord string) (string, bool) {
	zookeeperAddr := &addr
	log.Debugf("Connecting to Zookeeper at %s", *zookeeperAddr)

	conn, err := net.Dial("tcp", *zookeeperAddr)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Unable to open connection to Zookeeper")
		return "", false
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(timeoutSeconds * time.Second))
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to set timeout on Zookeeper connection")
		return "", false
	}

	log.WithFields(log.Fields{"command": fourLetterWord}).Debug("Sending four letter word")
	_, err = conn.Write([]byte(fourLetterWord))
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error sending command to Zookeeper")
		return "", false
	}
	scanner := bufio.NewScanner(conn)

	buffer := bytes.Buffer{}
	for scanner.Scan() {
		buffer.WriteString(scanner.Text() + "\n")
	}
	if err = scanner.Err(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error parsing response from Zookeeper")
		return "", false
	}
	log.Debug("Successfully retrieved reply")

	return buffer.String(), true
}
