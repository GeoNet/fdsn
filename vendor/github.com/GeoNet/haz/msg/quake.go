package msg

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/Golang-Ellipsoid/ellipsoid"
	"log"
	"math"
	"text/template"
	"time"
)

var (
	geo      ellipsoid.Ellipsoid
	alertAge = time.Duration(-60) * time.Minute
	nz       *time.Location
	t        = template.Must(template.New("eqNews").Parse(eqNews))
)

const (
	dutyTime    = "3:04 PM, 02/01/2006 MST"
	eqNewsNow   = "Mon 2 Jan 2006 at 3:04 pm"
	eqNewsUTC   = "2006/01/02 at 15:04:05"
	eqNewsLocal = "(MST):      Monday 2 Jan 2006 at 3:04 pm"
	tcoUrlLen   = len("https://t.co/7gZ0yUcmSx") // 09/06/2015 Twitter's t.co url sample (22 chars)
)

const eqNews = `                PRELIMINARY EARTHQUAKE REPORT

                      GeoNet Data Centre
                         GNS Science
                   Lower Hutt, New Zealand
                   http://www.geonet.org.nz

        Report Issued at: {{.Now}}


A likely felt earthquake has been detected by GeoNet; this is PRELIMINARY information only:

        Public ID:              {{.Q.PublicID}}
        Universal Time:         {{.UT}}
        Local Time {{.LT}}
        Latitude, Longitude:    {{.LL}}
        Location:               {{.Location}}
        Intensity:              {{.Intensity}} (MM{{.MMI}})
        Depth:                  {{ printf "%.f"  .Q.Depth}} km
        Magnitude:              {{ printf "%.1f"  .Q.Magnitude}}

Check for the LATEST information at http://www.geonet.org.nz/quakes/{{.Q.PublicID}}
`

type eqNewsD struct {
	Q         *Quake
	MMI       int
	Location  string
	Now       string
	TZ        string // timezone for the quake.
	UT        string // quake time in UTC
	LT        string // quake in local time
	LL        string // lon lat string
	Intensity string // word version of MMI
}

func init() {
	geo = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Kilometer, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingNotSymmetric)
	var err error
	nz, err = time.LoadLocation("Pacific/Auckland")
	if err != nil {
		log.Println("Error loading TZNZ carrying on with UTC")
		nz = time.UTC
	}
}

type Quake struct {
	PublicID              string
	Type                  string
	AgencyID              string
	ModificationTime      time.Time
	Time                  time.Time
	Latitude              float64
	Longitude             float64
	Depth                 float64
	DepthType             string
	MethodID              string
	EarthModelID          string
	EvaluationMode        string
	EvaluationStatus      string
	UsedPhaseCount        int
	UsedStationCount      int
	StandardError         float64
	AzimuthalGap          float64
	MinimumDistance       float64
	Magnitude             float64
	MagnitudeUncertainty  float64
	MagnitudeType         string
	MagnitudeStationCount int
	Site                  string
	err                   error
}

// Status returns the public status for the Quake referred to by q.
// Returns 'error' if q.Err() is not nil.
func (q *Quake) Status() string {
	if q.err != nil {
		return "error"
	}

	switch {
	case q.Type == "not existing":
		return "deleted"
	case q.Type == "duplicate":
		return "duplicate"
	case q.EvaluationMode == "manual":
		return "reviewed"
	case q.EvaluationStatus == "confirmed":
		return "reviewed"
	default:
		return "automatic"
	}
}

func (q *Quake) Quality() string {
	if q.err != nil {
		return "error"
	}

	status := q.Status()

	switch {
	case status == "reviewed":
		return "best"
	case status == "deleted":
		return "deleted"
	case q.UsedPhaseCount >= 20 && q.MagnitudeStationCount >= 10:
		return "good"
	default:
		return "caution"
	}
}

func (q *Quake) Err() error {
	return q.err
}

func (q *Quake) SetErr(err error) {
	q.err = err
}

func (q *Quake) RxLog() {
	if q.err != nil {
		return
	}

	log.Printf("Received quake %s", q.PublicID)
}

func (q *Quake) TxLog() {
	if q.err != nil {
		return
	}

	log.Printf("Sending quake %s", q.PublicID)
}

// MMI calculates the maximum Modificed Mercalli Intensity for the quake.
func (q *Quake) MMI() float64 {
	if q.err != nil {
		return -1. - 0
	}

	var w, m float64
	d := math.Abs(q.Depth)
	rupture := d

	if d < 100 {
		w = math.Min(0.5*math.Pow(10, q.Magnitude-5.39), 30.0)
		rupture = math.Max(d-0.5*w*0.85, 0.0)
	}

	if d < 70.0 {
		m = 4.40 + 1.26*q.Magnitude - 3.67*math.Log10(rupture*rupture*rupture+1634.691752)/3.0 + 0.012*d + 0.409
	} else {
		m = 3.76 + 1.48*q.Magnitude - 3.50*math.Log10(rupture*rupture*rupture)/3.0 + 0.0031*d
	}

	if m < 3.0 {
		m = -1.0
	}

	return m
}

// MMIDistance calculates the MMI at distance for New Zealand.  Distance and depth are in km.
func MMIDistance(distance, depth, mmi float64) float64 {
	// Minimum depth of 5 for numerical instability.
	d := math.Max(math.Abs(depth), 5.0)
	s := math.Hypot(d, distance)

	return math.Max(mmi-1.18*math.Log(s/d)-0.0044*(s-d), -1.0)
}

// MMIIntensity returns the string describing mmi.
func MMIIntensity(mmi float64) string {
	switch {
	case mmi >= 7:
		return "severe"
	case mmi >= 6:
		return "strong"
	case mmi >= 5:
		return "moderate"
	case mmi >= 4:
		return "light"
	case mmi >= 3:
		return "weak"
	default:
		return "unnoticeable"
	}
}

// IntensityMMI returns the minimum MMI for the instensity.
func IntensityMMI(Intensity string) float64 {
	switch Intensity {
	case "severe":
		return 7
	case "strong":
		return 6
	case "moderate":
		return 5
	case "light":
		return 4
	case "weak":
		return 3
	default:
		return -9
	}
}

// Closest returns the New Zealand LocalityQuake closest to the quake.
func (q *Quake) Closest() (loc LocalityQuake, err error) {
	loc, err = q.ClosestInRegion(NewZealand)
	return
}

// Closest returns the Region LocalityQuake closest to the quake.
func (q *Quake) ClosestInRegion(r RegionID) (loc LocalityQuake, err error) {
	if q.err != nil {
		err = q.err
		return
	}

	distance := 20000.0
	var bearing float64
	var locality Locality

	for _, l := range regions[r] {
		d, b := geo.To(l.Latitude, l.Longitude, q.Latitude, q.Longitude)
		if d < distance {
			distance = d
			locality = l
			bearing = b
		}
	}

	// ensure larger locality when distant quake.
	if distance > 300 && locality.size >= 2 {
		distance = 20000

		for _, l := range regions[r] {
			if l.size == 0 || l.size == 1 {
				d, b := geo.To(l.Latitude, l.Longitude, q.Latitude, q.Longitude)
				if d < distance {
					distance = d
					locality = l
					bearing = b
				}
			}
		}
	}

	loc.Locality = locality
	loc.Distance = distance
	loc.Bearing = bearing
	loc.MMIDistance = MMIDistance(distance, q.Depth, q.MMI())

	return loc, nil
}

/*
LocalitiesQuake returns localities in New Zealand that have an MMI at a distance >= minMMIDistance
for the quake.
*/
func (q Quake) Localities(minMMIDistance float64) (l []LocalityQuake) {
	if q.err != nil {
		return
	}

	mmi := q.MMI()

	for _, loc := range regions[NewZealand] {
		d, b := geo.To(loc.Latitude, loc.Longitude, q.Latitude, q.Longitude)

		mmid := MMIDistance(d, q.Depth, mmi)

		if mmid >= minMMIDistance {
			c := LocalityQuake{
				Locality:    loc,
				Distance:    d,
				Bearing:     b,
				MMIDistance: mmid,
			}

			l = append(l, c)
		}
	}

	return
}

// Returns true of the Quake is of high enough quality to consider for alerting.
//  false if not.
func (q *Quake) AlertQuality() bool {
	if q.err != nil {
		return false
	}

	switch {
	case q.Status() == "deleted":
		log.Printf("%s status deleted not suitable for alerting.", q.PublicID)
		return false
	case q.Status() == "duplicate":
		log.Printf("%s status duplicate not suitable for alerting.", q.PublicID)
		return false
	case q.Status() == "automatic" && (q.UsedPhaseCount < 20 || q.MagnitudeStationCount < 10):
		log.Printf("%s unreviewed with %d phases and %d magnitudes not suitable for alerting.", q.PublicID, q.UsedPhaseCount, q.MagnitudeStationCount)
		return false
	case q.Time.Before(time.Now().UTC().Add(alertAge)):
		log.Printf("%s to old for alerting", q.PublicID)
		return false
	}

	return true
}

// Publish returns true if the quake is suitable for publishing.
// site is either 'primary' or 'backup'.
func (q *Quake) Publish() bool {
	if q.err != nil {
		return false
	}

	p := true
	switch q.Site {
	case "primary", "":
		if q.Status() == "automatic" && !(q.Depth >= 0.1 && q.AzimuthalGap <= 320.0 && q.MinimumDistance <= 2.5) {
			p = false
			log.Printf("Not publising automatic quake %s with poor quality from primary site.", q.PublicID)
		}
	case "backup":
		if q.Status() == "automatic" {
			p = false
			log.Printf("Not publising unreviewed quake %s from backup site.", q.PublicID)
		}
	}
	return p
}

// AlertDuty returns alert = true and message formated if the quake is suitable for alerting the
// duty people, alert = false and empty message if not.
func (q *Quake) AlertDuty() (alert bool, message string) {
	if q.Err() != nil {
		return
	}

	if !q.AlertQuality() {
		return
	}

	mmi := q.MMI()

	if mmi >= 6 || q.Magnitude >= 4.5 {
		alert = true

		c, err := q.Closest()
		if err != nil {
			q.SetErr(err)
			return
		}

		// Eq Rpt: MAG 5.0, MM7, DEP 10, LOC 105 km N of White Island, TIME 08:33 AM, 26/02/2015
		message = fmt.Sprintf("Eq Rpt: MAG %.1f, MM%d, DEP %.f, LOC %s %s of %s, TIME %s",
			q.Magnitude,
			int(mmi),
			q.Depth,
			Distance(c.Distance),
			Compass(c.Bearing),
			c.Locality.Name,
			q.Time.In(nz).Format(dutyTime))
	}

	return
}

// AlertPIM returns alert = true and message formated if the quake is suitable for alerting the
// Pubilc Information people, alert = false and empty message if not.
func (q *Quake) AlertPIM() (alert bool, message string) {
	if q.Err() != nil {
		return
	}

	if !q.AlertQuality() {
		return
	}

	if q.Magnitude >= 6.0 {
		alert = true

		mmi := q.MMI()

		c, err := q.Closest()
		if err != nil {
			q.SetErr(err)
			return
		}

		// Eq Rpt: MAG 5.0, MM7, DEP 10, LOC 105 km N of White Island, TIME 08:33 AM, 26/02/2015
		message = fmt.Sprintf("Eq Rpt: MAG %.1f, MM%d, DEP %.f, LOC %s %s of %s, TIME %s",
			q.Magnitude,
			int(mmi),
			q.Depth,
			Distance(c.Distance),
			Compass(c.Bearing),
			c.Locality.Name,
			q.Time.In(nz).Format(dutyTime))
	}

	return
}

/*
AlertTwitter returns alert = true and message formatted for sending to twitter if
the quake is suitable for alerting and above the minMagnitude threshold.  alert = false
and message empty if not.
*/
func (q *Quake) AlertTwitter(minMagnitude float64) (alert bool, message string) {
	if q.Err() != nil {
		return
	}

	if !q.AlertQuality() {
		return
	}

	if q.Magnitude < minMagnitude {
		return
	}

	c, err := q.Closest()
	if err != nil {
		q.SetErr(err)
		return
	}

	if c.MMIDistance < 3.0 {
		return
	}

	alert = true

	// Quake 85 km east of Ruatoria, intensity moderate, approx. M3.6, depth 6 km http://geonet.org.nz/quakes/2011a868660 Fri Nov 18 2011 10:42 PM (NZDT)
	qUrl := fmt.Sprintf("http://geonet.org.nz/quakes/%s", q.PublicID)
	message = fmt.Sprintf("M%0.1f quake causing %s shaking near %s %s", q.Magnitude, MMIIntensity(c.MMIDistance), c.Locality.Name, qUrl)

	// Make sure we'll only send message less than 140 chars (after url shortened with t.co)
	t := len(message) - len(qUrl) + tcoUrlLen - 140
	if t > 0 {
		message = message[0 : len(message)-t]
		log.Println("WARNING: Twitter message truncated", t, "chars to:", message)
	}

	return
}

func (q *Quake) AlertUAPush() (message string, tags []string) {
	if q.Err() != nil {
		return
	}

	if !q.AlertQuality() {
		return
	}

	c, err := q.Closest()
	if err != nil {
		q.SetErr(err)
		return
	}

	if c.MMIDistance < 3.0 {
		return
	}

	tags = q.uaTags()
	message = fmt.Sprintf("M%0.1f quake causing %s shaking near %s", q.Magnitude, MMIIntensity(c.MMIDistance), c.Locality.Name)

	return
}

func (q *Quake) AlertEqNews() (alert bool, subject, body string) {
	if q.Err() != nil {
		return
	}

	if !q.AlertQuality() {
		return
	}

	mmi := q.MMI()

	c, err := q.Closest()
	if err != nil {
		q.SetErr(err)
		return
	}

	if mmi >= 7.0 || c.MMIDistance >= 3.5 {
		alert = true

		// NZ EQ: M3.5, weak intensity, 5km deep, 20 km N of Reefton
		subject = fmt.Sprintf("NZ EQ: M%.1f, %s intensity, %.fkm deep, %s %s of %s",
			q.Magnitude,
			MMIIntensity(mmi),
			q.Depth,
			Distance(c.Distance),
			Compass(c.Bearing),
			c.Locality.Name)

	}

	buf := new(bytes.Buffer)

	err = t.ExecuteTemplate(buf, "eqNews", &eqNewsD{
		Q:         q,
		MMI:       int(mmi),
		Location:  c.Location(),
		Now:       time.Now().In(nz).Format(eqNewsNow),
		UT:        q.Time.Format(eqNewsUTC),
		LT:        q.Time.In(nz).Format(eqNewsLocal),
		LL:        q.eqNewsLonLat(),
		Intensity: MMIIntensity(mmi),
	})
	if err != nil {
		q.SetErr(err)
		alert = false
		return
	}

	body = buf.String()

	return

}

func Distance(km float64) string {
	s := "Within 5 km of"

	d := math.Floor(km / 5.0)
	if d > 0 {
		s = fmt.Sprintf("%.f km", d*5)
	}
	return s
}

func (q *Quake) eqNewsLonLat() string {
	var lon, lat string

	switch q.Longitude < 0.0 {
	case true:
		lon = fmt.Sprintf("%.2fW", q.Longitude*-1.0)
	case false:
		lon = fmt.Sprintf("%.2fE", q.Longitude)
	}

	switch q.Latitude < 0.0 {
	case true:
		lat = fmt.Sprintf("%.2fS", q.Latitude*-1.0)
	case false:
		lat = fmt.Sprintf("%.2fN", q.Latitude)
	}

	// 41.94S, 171.86E
	return lat + ", " + lon
}
