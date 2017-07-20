// ddoghttp collects metrics for messaging applications.
// If the env var DDOG_API_KEY is set then metrics are sent to the data dog api
// otherwise they are logged.
//
// Import for side effects.
package ddoghttp

import (
	"bytes"
	"encoding/json"
	"github.com/GeoNet/fdsn/internal/platform/metrics"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const dogUrl = "https://app.datadoghq.com/api/v1/series"

var apiKey = os.Getenv("DDOG_API_KEY")
var client = &http.Client{}

type point [2]float32

// metric is for sending metrics to datadog.
type metric struct {
	Metric string  `json:"metric"`
	Points []point `json:"points"`
	Type   string  `json:"type"`
	Host   string  `json:"host"`
}

type series struct {
	Series []metric `json:"series"`
}

func init() {
	s := os.Args[0]
	appName := strings.Replace(s[strings.LastIndex(s, "/")+1:], "-", "_", -1)

	hostName, err := os.Hostname()
	if err != nil {
		log.Println("error finding hostname " + err.Error())
	}

	if apiKey == "" {
		log.Print("empty env var DDOG_API_KEY metrics will be logged")
	}

	go func() {
		var c metrics.HttpCounters
		var m runtime.MemStats

		ticker := time.NewTicker(time.Second * 60).C
		var err error

		for {
			select {
			case <-ticker:
				metrics.ReadHttpCounters(&c)
				runtime.ReadMemStats(&m)

				if apiKey != "" {
					err = dog(hostName, appName, m, metrics.ReadTimers(), c)
					if err != nil {
						log.Printf("error sending metrics to datadog for %s %s %s", hostName, appName, err.Error())
					}
				} else {
					log.Printf("%s %s", hostName, appName)
					log.Printf("%+v", m)
					log.Printf("%+v", metrics.ReadTimers())
					log.Printf("%+v", c)
				}
			}
		}
	}()
}

func dog(hostName, appName string, m runtime.MemStats, t []metrics.TimerStats, c metrics.HttpCounters) error {
	now := float32(time.Now().Unix())

	var series = series{Series: []metric{
		{
			Metric: appName + ".mem.sys",
			Points: []point{[2]float32{now, float32(m.Sys)}},
			Type:   "gauge",
			Host:   hostName,
		},
		{
			Metric: appName + ".mem.heap.sys",
			Points: []point{[2]float32{now, float32(m.HeapSys)}},
			Type:   "gauge",
			Host:   hostName,
		},
		{
			Metric: appName + ".mem.heap.alloc",
			Points: []point{[2]float32{now, float32(m.HeapAlloc)}},
			Type:   "gauge",
			Host:   hostName,
		},
		{
			Metric: appName + ".mem.heap.objects",
			Points: []point{[2]float32{now, float32(m.HeapObjects)}},
			Type:   "gauge",
			Host:   hostName,
		},
		{
			Metric: appName + ".goroutines",
			Points: []point{[2]float32{now, float32(runtime.NumGoroutine())}},
			Type:   "gauge",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.req",
			Points: []point{[2]float32{now, float32(c.Request)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.200",
			Points: []point{[2]float32{now, float32(c.StatusOK)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.400",
			Points: []point{[2]float32{now, float32(c.StatusBadRequest)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.404",
			Points: []point{[2]float32{now, float32(c.StatusNotFound)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.401",
			Points: []point{[2]float32{now, float32(c.StatusUnauthorized)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.500",
			Points: []point{[2]float32{now, float32(c.StatusInternalServerError)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".http.503",
			Points: []point{[2]float32{now, float32(c.StatusServiceUnavailable)}},
			Type:   "counter",
			Host:   hostName,
		},
	},
	}

	for _, v := range t {
		series.Series = append(series.Series, metric{
			Metric: appName + ".timer." + v.ID + ".95percentile",
			Points: []point{[2]float32{now, float32(v.Percentile95)}},
			Type:   "gauge",
			Host:   hostName,
		})
		series.Series = append(series.Series, metric{
			Metric: appName + ".timer." + v.ID + ".count",
			Points: []point{[2]float32{now, float32(v.Count)}},
			Type:   "gauge",
			Host:   hostName,
		})
	}

	b, err := json.Marshal(&series)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", dogUrl, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-type", "application/json")

	q := req.URL.Query()
	q.Add("api_key", apiKey)

	req.URL.RawQuery = q.Encode()

	var res *http.Response

	for tries := 0; time.Now().Before(time.Now().Add(time.Second * 30)); tries++ {
		if res, err = client.Do(req); err == nil {
			if res != nil && res.StatusCode == 202 {
				break
			} else {
				err = errors.Errorf("Non 202 code from datadog: %d", res.StatusCode)
				break
			}
		}
		// non nil connection error, sleep and try again
		time.Sleep(time.Second << uint(tries))
	}
	if res != nil {
		res.Body.Close()
	}

	return err
}
