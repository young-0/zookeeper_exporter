package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	// "github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)
type HandleFnc func(http.ResponseWriter,*http.Request)

func main() {
	var (
		bindAddr  = flag.String("bind-addr", ":8080", "bind address for the metrics server")
		logFormat = flag.String("log-format", getEnv("LOG_FORMAT", "txt"), "Log format, valid options are txt and json")
		isDebug   = flag.Bool("debug", getEnvBool("LOG_DEBUG", false), "Output verbose debug information")
	)
	flag.Parse()
	// 日志格式设置
	switch *logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}
	if *isDebug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Enabling debug output")
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Info("Starting zookeeper_exporter")

	// 构造函数创造 ZookeeperCollector
	addr := "127.0.0.1:2181"
	opt := Options{
		Registry: prometheus.NewRegistry()}
	exp := NewZookeeperCollector(addr, &opt)

	go serveMetrics(bindAddr, exp)
	// 等待停止信号
	exitChannel := make(chan os.Signal)
	signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	exitSignal := <-exitChannel
	log.WithFields(log.Fields{"signal": exitSignal}).Infof("Caught %s signal, exiting", exitSignal)
}

func serveMetrics(bindAddr *string, exp *zookeeperCollector) {
	log.Infof("Starting metric http endpoint on %s", *bindAddr)
	log.Fatal(http.ListenAndServe(*bindAddr, exp))
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html>
		<head><title>Zookeeper Exporter</title></head>
		<body>
		<h1>Zookeeper Exporter</h1>
		<p>/scrape?target=ip:port</p>
		</body>
		</html>`))
}
func getEnv(key string, defaultVal string) string {
	if envVal, ok := os.LookupEnv(key); ok {
		return envVal
	}
	return defaultVal
}
func getEnvBool(key string, defaultVal bool) bool {
	if envVal, ok := os.LookupEnv(key); ok {
		envBool, err := strconv.ParseBool(envVal)
		if err == nil {
			return envBool
		}
	}
	return defaultVal
}
func logPanics(function HandleFnc) HandleFnc {
	return func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				log.Printf("[%v] caught panic: %v", request.RemoteAddr, x)
				//默认出现 panic 只会记录日志，页面就是一个无任何输出的白页面，
				// 可以给页面一个错误信息，如下面的示例返回了一个 500
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		function(writer, request)
	}
}
