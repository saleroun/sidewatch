/*
Custom exporters requires 4 stubs:
 a. A structure with member variables
 b. A factory method that returns the structure
 c. Describe function
 d. Collect function

gauge vs counter :
 Gauges are suitable for measuring instantaneous values
 that can both increase and decrease,
 while Counters are used for tracking cumulative counts that only increase over time.
 The choice between Gauge and Counter depends on the nature of the metric
 you want to monitor and the behavior you want to capture.

cpu and memory profiling:
# uncomment lines [90 - 101] if you want to use profiling features.
1. run your program then monitor cpu and memory profiling whith commands bellow:

	go tool pprof cpu_profile.pprof
	go tool pprof memory_profile.pprof

author: Saleroun
*/

//----------------------------------------------------------------

package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	_ "github.com/taosdata/driver-go/v3/taosRestful"

	"github.com/ghodss/yaml"
	"github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)

var config Config

const (
	collector = "sidewatch"
)

type ServerStatus struct {
	URL        string
	StatusCode int
	Status     string
}
type PingResult struct {
	Hostname string
	Stat     *ping.Statistics
	Err      error
}

type Config struct {
	// Container_name string
	ContaierName string
	Metrics      map[string]struct {
		Description string
		Labels      []string
		Type        string
		Url         string
		Timeout     int
		metricDesc  *prometheus.Desc
	}
}

var Data = make(map[string]string)

var LabelsVal = make([]string, 2)

type QueryCollector struct{}

func main() {

	var err error
	var configFile, bind string

	flag.StringVar(&configFile, "config", "config.yml", "configuration file")
	flag.StringVar(&bind, "bind", "0.0.0.0:9100", "bind")
	flag.Parse()
	// cpu, _ := os.Create("cpu_profile.pprof")
	// defer cpu.Close()

	// // Start CPU profiling
	// pprof.StartCPUProfile(cpu)
	// defer pprof.StopCPUProfile()

	// f, _ := os.Create("memory_profile.pprof")
	// defer f.Close()

	// // Start memory profiling
	// pprof.WriteHeapProfile(f)

	var b []byte
	if b, err = ioutil.ReadFile(configFile); err != nil {
		log.Errorf("Failed to read config file: %s", err)
		os.Exit(1)
	}

	// Load yaml
	if err := yaml.Unmarshal(b, &config); err != nil {
		log.Errorf("Failed to load config: %s", err)
		os.Exit(1)
	}

	if ok := os.Getenv("NODE_NAME"); ok != "" {
		Data["container"] = ok
		LabelsVal[0] = Data["container"]
	} else {
		log.Fatal("NODE_NAME is not valid or nil")
	}
	log.Infof("Regist version collector - %s", collector)
	prometheus.Register(version.NewCollector(collector))
	prometheus.Register(&QueryCollector{})

	log.Infof("HTTP handler path - %s", "/metrics")
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h := promhttp.HandlerFor(prometheus.Gatherers{
			prometheus.DefaultGatherer,
		}, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})

	// start server
	log.Infof("Starting http server - %s", bind)
	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Errorf("Failed to start http server: %s", err)
	}

}

func (e *QueryCollector) Describe(ch chan<- *prometheus.Desc) {
	for metricName, metric := range config.Metrics {
		metric.metricDesc = prometheus.NewDesc(
			prometheus.BuildFQName(collector, "", metricName),
			metric.Description,
			metric.Labels, nil,
		)
		config.Metrics[metricName] = metric
		log.Infof("metric description for \"%s\" registerd", metricName)
	}
}

func (e *QueryCollector) Collect(ch chan<- prometheus.Metric) {
	for name, metric := range config.Metrics {

		switch metric.Labels[1] {
		case "http":
			func() {
				hStat, err := HttpCheck(metric.Url, metric.Timeout)
				if hStat == nil || err != nil {
					hStat.StatusCode = 0
				}
				Data[metric.Labels[1]] = metric.Url
				LabelsVal[1] = Data[metric.Labels[1]]
				switch strings.ToLower(metric.Type) {
				case "counter":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.CounterValue, float64(hStat.StatusCode), LabelsVal...)
				case "gauge":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.GaugeValue, float64(hStat.StatusCode), LabelsVal...)
				default:
					log.Errorf("Fail to add metric for %s: %s is not valid type", name, metric.Type)
				}

			}()

		case "amqp":
			func() {
				amStat := isRabbitMQHealthy(metric.Url)
				// log.Info(name, "Collecting amqp")
				Data[metric.Labels[1]] = "rabbitmq"
				LabelsVal[1] = Data[metric.Labels[1]]
				switch strings.ToLower(metric.Type) {
				case "counter":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.CounterValue, amStat, LabelsVal...)
				case "gauge":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.GaugeValue, amStat, LabelsVal...)
				default:
					log.Errorf("Fail to add metric for %s: %s is not valid type", name, metric.Type)
				}
			}()
		case "mongo":
			func() {
				mgStat := isMongoDBHealthy(metric.Url, metric.Timeout)
				// log.Info(name, "Collecting mongo")
				Data[metric.Labels[1]] = "mongodb"
				LabelsVal[1] = Data[metric.Labels[1]]
				switch strings.ToLower(metric.Type) {
				case "counter":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.CounterValue, mgStat, LabelsVal...)
				case "gauge":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.GaugeValue, mgStat, LabelsVal...)
				default:
					log.Errorf("Fail to add metric for %s: %s is not valid type", name, metric.Type)
				}
			}()
		case "redis":
			func() {
				rdStat := isRedisHealthy(metric.Url, metric.Timeout)
				// log.Info(name, "Collecting redis")
				Data[metric.Labels[1]] = "redis"
				LabelsVal[1] = Data[metric.Labels[1]]
				switch strings.ToLower(metric.Type) {
				case "counter":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.CounterValue, rdStat, LabelsVal...)
				case "gauge":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.GaugeValue, rdStat, LabelsVal...)
				default:
					log.Errorf("Fail to add metric for %s: %s is not valid type", name, metric.Type)
				}
			}()
		case "taos":
			func() {
				tdStat := isTDengineHealthy(metric.Url, metric.Timeout)
				// log.Info(name, "Collecting td")
				Data[metric.Labels[1]] = "tdengine"
				LabelsVal[1] = Data[metric.Labels[1]]
				switch strings.ToLower(metric.Type) {
				case "counter":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.CounterValue, tdStat, LabelsVal...)
				case "gauge":
					ch <- prometheus.MustNewConstMetric(metric.metricDesc, prometheus.GaugeValue, tdStat, LabelsVal...)
				default:
					log.Errorf("Fail to add metric for %s: %s is not valid type", name, metric.Type)
				}
			}()
		default:
			log.Warningln("There is no valid label ", metric.Labels[1])
		}
		// fmt.Println("---------------------------------")
	}
}
