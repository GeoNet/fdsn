package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// DataDogHttp initiates collection of HTTP and system metrics every 60s.
// if apiKey is non zero metrics are send to Data Dog otherwise they are logged
// using logger.  Errors are logged with logger.
// If logger is nil log messages are discarded.
func DataDogHttp(apiKey, hostName, appName string, logger Logger) {
	if logger == nil {
		logger = discarder{}
	}

	if apiKey == "" {
		logger.Printf("empty env var DDOG_API_KEY metrics will be logged")
	}

	go func() {
		var c HttpCounters
		var m runtime.MemStats

		ticker := time.NewTicker(time.Second * 60).C
		var err error

		for range ticker {
			ReadHttpCounters(&c)
			runtime.ReadMemStats(&m)

			if apiKey != "" {
				err = dogHttp(apiKey, hostName, appName, m, ReadTimers(), c)
				if err != nil {
					logger.Printf("error sending metrics to datadog for %s %s %s", hostName, appName, err.Error())
				}
			} else {
				logger.Printf("%s %s", hostName, appName)
				logger.Printf("%+v", m)
				logger.Printf("%+v", ReadTimers())
				logger.Printf("%+v", c)
			}
		}
	}()
}

func dogHttp(apiKey, hostName, appName string, m runtime.MemStats, t []TimerStats, c HttpCounters) error {
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
		{
			Metric: appName + ".http.written",
			Points: []point{[2]float32{now, float32(c.Written)}},
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
				err = fmt.Errorf("non 202 code from datadog: %d", res.StatusCode)
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
