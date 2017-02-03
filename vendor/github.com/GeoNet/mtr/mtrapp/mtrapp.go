/*
mtrapp for gathering application metrics.

init initalizes the collection and sending of metrics once per minute if the environment var
MTR_SERVER MTR_USER and MTR_KEY are all non zero.
ApplicationID and InstanceID default to the executable and host names.  These can be set with
the environment var MTR_APPLICATIONID and MTR_INSTANCEID.

Import for side effects  to collect memory and runtime metrics only.
*/
package mtrapp

import (
	"github.com/GeoNet/mtr/internal"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const timeout = 3 * time.Minute

var (
	appName           string
	instanceID        string
	server, user, key string
	client            = &http.Client{}
)

func init() {
	appName = os.Getenv("MTR_APPLICATIONID")
	instanceID = os.Getenv("MTR_INSTANCEID")
	server = os.Getenv("MTR_SERVER")
	user = os.Getenv("MTR_USER")
	key = os.Getenv("MTR_KEY")

	if appName == "" {
		s := os.Args[0]
		appName = s[strings.LastIndex(s, "/")+1:]
	}

	if instanceID == "" {
		var err error
		instanceID, err = os.Hostname()
		if err != nil {
			log.Println("error finding hostname " + err.Error())
		}
	}

	switch "" {
	case server, user, key:
		log.Println("no mtr credentials, metrics will be dropped.")
	default:
		go func() {
			var mem runtime.MemStats

			ticker := time.NewTicker(time.Minute).C

			var last = time.Now().UTC()
			var now time.Time

			for {
				select {
				case m := <-timers:
					count[m.id]++
					sum[m.id] += m.taken
					taken[m.id] = append(taken[m.id], m.taken)
				case <-ticker:
					now = time.Now().UTC()

					runtime.ReadMemStats(&mem)

					go sendMetric(internal.MemSys, now, int64(mem.Sys))
					go sendMetric(internal.MemHeapAlloc, now, int64(mem.HeapAlloc))
					go sendMetric(internal.MemHeapSys, now, int64(mem.HeapSys))
					go sendMetric(internal.MemHeapObjects, now, int64(mem.HeapObjects))
					go sendMetric(internal.Routines, now, int64(runtime.NumGoroutine()))

					// assume that retrieving values from the counters is fast
					// enough that we don't need a time for each one.
					for i := range counters {
						currVal[i] = counters[i].value()
					}

					for i := range counters {
						if v := currVal[i] - lastVal[i]; v > 0 {
							go sendCount(counters[i].id, last, int(v))
						}
					}

					for i := range counters {
						lastVal[i] = currVal[i]
					}

					for k, v := range count {
						a := sum[k] / v
						f := percentile(0.5, taken[k])
						n := percentile(0.9, taken[k])

						go sendTimer(k, last, v, a, f, n)

						delete(taken, k)
						delete(sum, k)
						delete(count, k)
					}

					last = now
				}
			}
		}()
	}
}

// calculates the kth percentile of v
func percentile(k float64, v []int) (value int) {
	if !sort.IntsAreSorted(v) {
		sort.Ints(v)
	}

	p := k * float64(len(v))

	if p != math.Trunc(p) {
		idx := int(math.Ceil(p))
		if idx <= len(v) {
			value = v[int(math.Ceil(p))-1]
		}
	} else {
		idx := int(math.Trunc(p))
		if idx < len(v) {
			value = int((v[idx-1] + v[idx]) / 2)
		}
	}

	return
}

func sendMetric(typeID internal.ID, t time.Time, value int64) {
	var req *http.Request
	var res *http.Response
	var err error

	if req, err = http.NewRequest("PUT", server+"/application/metric", nil); err != nil {
		// TODO log error ?
		return
	}

	req.SetBasicAuth(user, key)

	q := req.URL.Query()
	q.Add("applicationID", appName)
	q.Add("instanceID", instanceID)
	q.Add("typeID", strconv.Itoa(int(typeID)))
	q.Add("time", t.Format(time.RFC3339))
	q.Add("value", strconv.FormatInt(value, 10))
	req.URL.RawQuery = q.Encode()

	deadline := time.Now().Add(timeout)

	for tries := 0; time.Now().Before(deadline); tries++ {
		if res, err = client.Do(req); err == nil {
			if res != nil && res.StatusCode != 200 {
				log.Printf("Non 200 code from metrics: %d", res.StatusCode)
			}
			break
		}
		log.Printf("server not responding (%s); backing off and retrying...", err)
		time.Sleep(time.Second << uint(tries))
	}
	if res != nil {
		res.Body.Close()
	}
}

func sendCount(typeID internal.ID, t time.Time, count int) {
	var req *http.Request
	var res *http.Response
	var err error

	if req, err = http.NewRequest("PUT", server+"/application/counter", nil); err != nil {
		// TODO log error ?
		return
	}

	req.SetBasicAuth(user, key)

	q := req.URL.Query()
	q.Add("applicationID", appName)
	q.Add("instanceID", instanceID)
	q.Add("typeID", strconv.Itoa(int(typeID)))
	q.Add("time", t.Format(time.RFC3339))
	q.Add("count", strconv.Itoa(count))
	req.URL.RawQuery = q.Encode()

	deadline := time.Now().Add(timeout)

	for tries := 0; time.Now().Before(deadline); tries++ {
		if res, err = client.Do(req); err == nil {
			if res != nil && res.StatusCode != 200 {
				log.Printf("Non 200 code from metrics: %d", res.StatusCode)
			}
			break
		}
		log.Printf("server not responding (%s); backing off and retrying...", err)
		time.Sleep(time.Second << uint(tries))
	}
	if res != nil {
		res.Body.Close()
	}
}

func sendTimer(sourceID string, t time.Time, count, average, fifty, ninety int) {
	var req *http.Request
	var res *http.Response
	var err error

	if req, err = http.NewRequest("PUT", server+"/application/timer", nil); err != nil {
		// TODO log error ?
		return
	}

	req.SetBasicAuth(user, key)

	q := req.URL.Query()
	q.Add("applicationID", appName)
	q.Add("instanceID", instanceID)
	q.Add("sourceID", sourceID)
	q.Add("time", t.Format(time.RFC3339))
	q.Add("count", strconv.Itoa(count))
	q.Add("average", strconv.Itoa(average))
	q.Add("fifty", strconv.Itoa(fifty))
	q.Add("ninety", strconv.Itoa(ninety))
	req.URL.RawQuery = q.Encode()

	deadline := time.Now().Add(timeout)

	for tries := 0; time.Now().Before(deadline); tries++ {
		if res, err = client.Do(req); err == nil {
			if res != nil && res.StatusCode != 200 {
				log.Printf("Non 200 code from metrics: %d", res.StatusCode)
			}
			break
		}
		log.Printf("server not responding (%s); backing off and retrying...", err)
		time.Sleep(time.Second << uint(tries))
	}
	if res != nil {
		res.Body.Close()
	}
}
