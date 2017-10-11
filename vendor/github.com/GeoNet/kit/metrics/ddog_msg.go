package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const dogUrl = "https://app.datadoghq.com/api/v1/series"

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

// AppName returns the application name.
func AppName() string {
	s := os.Args[0]
	return strings.Replace(s[strings.LastIndex(s, "/")+1:], "-", "_", -1)
}

// Hostname returns the hostname (can be empty).
func HostName() string {
	h, _ := os.Hostname()
	return h
}

// DataDogMsg initiates collection of messaging and system metrics every 60s.
// if apiKey is non zero metrics are send to Data Dog otherwise they are logged
// using logger.  Errors are logged with logger.
// If logger is nil log messages are discarded
func DataDogMsg(apiKey, hostName, appName string, logger Logger) {
	if logger == nil {
		logger = discarder{}
	}

	if apiKey == "" {
		logger.Printf("empty apiKey metrics will be logged")
	}

	go func() {
		var c MsgCounters
		var m runtime.MemStats

		ticker := time.NewTicker(time.Second * 60).C
		var err error

		for {
			select {
			case <-ticker:
				ReadMsgCounters(&c)
				runtime.ReadMemStats(&m)

				if apiKey != "" {
					err = dogMsg(apiKey, hostName, appName, m, ReadTimers(), c)
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
		}
	}()
}

func dogMsg(apiKey, hostName, appName string, m runtime.MemStats, t []TimerStats, c MsgCounters) error {
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
			Metric: appName + ".msg.rx",
			Points: []point{[2]float32{now, float32(c.Rx)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".msg.tx",
			Points: []point{[2]float32{now, float32(c.Tx)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".msg.proc",
			Points: []point{[2]float32{now, float32(c.Proc)}},
			Type:   "counter",
			Host:   hostName,
		},
		{
			Metric: appName + ".msg.err",
			Points: []point{[2]float32{now, float32(c.Err)}},
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
