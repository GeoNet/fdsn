package wgs84

import (
	"fmt"
	"math"
)

type Locality struct {
	Name                string
	Longitude, Latitude float64
	Distance            float64
	Bearing             float64
}

type ByDistance []Locality

func (a ByDistance) Len() int           { return len(a) }
func (a ByDistance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDistance) Less(i, j int) bool { return a[i].Distance < a[j].Distance }

var nz = []struct {
	name                string
	longitude, latitude float64
}{
	{name: `Auckland`, longitude: 174.77, latitude: -36.85},
	{name: `Cambridge`, longitude: 175.47, latitude: -37.88},
	{name: `Cape Reinga`, longitude: 172.68, latitude: -34.43},
	{name: `Hamilton`, longitude: 175.28, latitude: -37.78},
	{name: `Kaitaia`, longitude: 173.27, latitude: -35.12},
	{name: `Kawhia`, longitude: 174.82, latitude: -38.07},
	{name: `Pukekohe`, longitude: 174.9, latitude: -37.2},
	{name: `Te Aroha`, longitude: 175.7, latitude: -37.53},
	{name: `Te Awamutu`, longitude: 175.33, latitude: -38.02},
	{name: `Thames`, longitude: 175.55, latitude: -37.15},
	{name: `Whangamata`, longitude: 175.87, latitude: -37.22},
	{name: `Whangarei`, longitude: 174.32, latitude: -35.72},
	{name: `Whitianga`, longitude: 175.7, latitude: -36.82},
	{name: `Murupara`, longitude: 176.7, latitude: -38.45},
	{name: `Ohakune`, longitude: 175.42, latitude: -39.42},
	{name: `Opotiki`, longitude: 177.28, latitude: -38.02},
	{name: `Rotorua`, longitude: 176.23, latitude: -38.13},
	{name: `Taihape`, longitude: 175.8, latitude: -39.68},
	{name: `Taupo`, longitude: 176.08, latitude: -38.7},
	{name: `Tauranga`, longitude: 176.17, latitude: -37.68},
	{name: `Tokoroa`, longitude: 175.87, latitude: -38.23},
	{name: `Turangi`, longitude: 175.8, latitude: -39},
	{name: `Whakatane`, longitude: 176.98, latitude: -37.97},
	{name: `White Island`, longitude: 177.18, latitude: -37.52},
	{name: `Gisborne`, longitude: 178.02, latitude: -38.67},
	{name: `Matawai`, longitude: 177.53, latitude: -38.35},
	{name: `Ruatoria`, longitude: 178.32, latitude: -37.88},
	{name: `Te Araroa`, longitude: 178.37, latitude: -37.63},
	{name: `Te Kaha`, longitude: 177.68, latitude: -37.75},
	{name: `Tokomaru Bay`, longitude: 178.32, latitude: -38.13},
	{name: `Tolaga Bay`, longitude: 178.3, latitude: -38.37},
	{name: `Hastings`, longitude: 176.85, latitude: -39.65},
	{name: `Napier`, longitude: 176.9, latitude: -39.5},
	{name: `Waipukurau`, longitude: 176.55, latitude: -40},
	{name: `Wairoa`, longitude: 177.42, latitude: -39.05},
	{name: `Hawera`, longitude: 174.28, latitude: -39.58},
	{name: `Mokau`, longitude: 174.62, latitude: -38.7},
	{name: `New Plymouth`, longitude: 174.07, latitude: -39.07},
	{name: `Opunake`, longitude: 173.85, latitude: -39.45},
	{name: `Stratford`, longitude: 174.28, latitude: -39.35},
	{name: `Taumarunui`, longitude: 175.27, latitude: -38.88},
	{name: `Te Kuiti`, longitude: 175.17, latitude: -38.33},
	{name: `Waverley`, longitude: 174.63, latitude: -39.77},
	{name: `Blenheim`, longitude: 173.95, latitude: -41.52},
	{name: `Castlepoint`, longitude: 176.22, latitude: -40.9},
	{name: `Dannevirke`, longitude: 176.1, latitude: -40.2},
	{name: `Eketahuna`, longitude: 175.7, latitude: -40.65},
	{name: `Feilding`, longitude: 175.57, latitude: -40.23},
	{name: `French Pass`, longitude: 173.83, latitude: -40.93},
	{name: `Hunterville`, longitude: 175.57, latitude: -39.93},
	{name: `Levin`, longitude: 175.28, latitude: -40.62},
	{name: `Martinborough`, longitude: 175.45, latitude: -41.22},
	{name: `Masterton`, longitude: 175.65, latitude: -40.95},
	{name: `Palmerston North`, longitude: 175.62, latitude: -40.37},
	{name: `Paraparaumu`, longitude: 175, latitude: -40.92},
	{name: `Picton`, longitude: 174, latitude: -41.3},
	{name: `Pongaroa`, longitude: 176.18, latitude: -40.55},
	{name: `Porangahau`, longitude: 176.62, latitude: -40.3},
	{name: `Seddon`, longitude: 174.07, latitude: -41.67},
	{name: `Wellington`, longitude: 174.77, latitude: -41.28},
	{name: `Lower Hutt`, longitude: 174.91, latitude: -41.21},
	{name: `Upper Hutt`, longitude: 175.07, latitude: -41.12},
	{name: `Porirua`, longitude: 174.84, latitude: -41.13},
	{name: `Whanganui`, longitude: 175.05, latitude: -39.93},
	{name: `Arthur's Pass`, longitude: 171.57, latitude: -42.95},
	{name: `Collingwood`, longitude: 172.68, latitude: -40.68},
	{name: `Greymouth`, longitude: 171.2, latitude: -42.45},
	{name: `Haast`, longitude: 169.05, latitude: -43.88},
	{name: `Hokitika`, longitude: 170.97, latitude: -42.72},
	{name: `Karamea`, longitude: 172.12, latitude: -41.25},
	{name: `Motueka`, longitude: 173.02, latitude: -41.12},
	{name: `Mount Cook`, longitude: 170.1, latitude: -43.73},
	{name: `Murchison`, longitude: 172.33, latitude: -41.8},
	{name: `Nelson`, longitude: 173.28, latitude: -41.27},
	{name: `Reefton`, longitude: 171.87, latitude: -42.12},
	{name: `St Arnaud`, longitude: 172.85, latitude: -41.8},
	{name: `Westport`, longitude: 171.6, latitude: -41.75},
	{name: `Akaroa`, longitude: 172.97, latitude: -43.82},
	{name: `Amberley`, longitude: 172.73, latitude: -43.17},
	{name: `Ashburton`, longitude: 171.75, latitude: -43.9},
	{name: `Cheviot`, longitude: 173.27, latitude: -42.82},
	{name: `Christchurch`, longitude: 172.63, latitude: -43.53},
	{name: `Culverden`, longitude: 172.85, latitude: -42.78},
	{name: `Fairlie`, longitude: 170.83, latitude: -44.1},
	{name: `Geraldine`, longitude: 171.23, latitude: -44.1},
	{name: `Hanmer Springs`, longitude: 172.83, latitude: -42.52},
	{name: `Kaikoura`, longitude: 173.68, latitude: -42.4},
	{name: `Methven`, longitude: 171.65, latitude: -43.63},
	{name: `Oxford`, longitude: 172.2, latitude: -43.3},
	{name: `Timaru`, longitude: 171.25, latitude: -44.4},
	{name: `Twizel`, longitude: 170.1, latitude: -44.27},
	{name: `Waimate`, longitude: 171.05, latitude: -44.73},
	{name: `Milford Sound`, longitude: 167.93, latitude: -44.68},
	{name: `Queenstown`, longitude: 168.67, latitude: -45.03},
	{name: `Te Anau`, longitude: 167.72, latitude: -45.42},
	{name: `Alexandra`, longitude: 169.38, latitude: -45.25},
	{name: `Balclutha`, longitude: 169.73, latitude: -46.23},
	{name: `Dunedin`, longitude: 170.5, latitude: -45.88},
	{name: `Gore`, longitude: 168.93, latitude: -46.1},
	{name: `Invercargill`, longitude: 168.37, latitude: -46.42},
	{name: `Lumsden`, longitude: 168.45, latitude: -45.73},
	{name: `Oamaru`, longitude: 170.97, latitude: -45.1},
	{name: `Palmerston`, longitude: 170.72, latitude: -45.48},
	{name: `Ranfurly`, longitude: 170.1, latitude: -45.13},
	{name: `Roxburgh`, longitude: 169.32, latitude: -45.55},
	{name: `Snares Islands`, longitude: 166.6, latitude: -48.02},
	{name: `Tuatapere`, longitude: 167.68, latitude: -46.13},
	{name: `Wanaka`, longitude: 169.13, latitude: -44.7},
}

// ClosestNZ returns the closest New Zealand locality to the input point.
// Locality.Bearing is from the Locality to the input point.
func ClosestNZ(latitude, longitude float64) (Locality, error) {
	distance := math.MaxFloat64
	var bearing float64
	var closest int

	for i := range nz {
		d, b, err := DistanceBearing(nz[i].latitude, nz[i].longitude, latitude, longitude)
		if err != nil {
			return Locality{}, err
		}

		if d < distance {
			distance = d
			bearing = b
			closest = i
		}
	}

	return Locality{Longitude: nz[closest].longitude, Latitude: nz[closest].latitude, Name: nz[closest].name, Bearing: bearing, Distance: distance}, nil
}

// LocalitiesNZ returns New Zealand localities for the input point.
// Locality.Bearing is from the Locality to the input point.
func LocalitiesNZ(latitude, longitude float64) ([]Locality, error) {
	var l []Locality

	for i := range nz {
		d, b, err := DistanceBearing(nz[i].latitude, nz[i].longitude, latitude, longitude)
		if err != nil {
			return []Locality{}, err
		}

		l = append(l, Locality{Longitude: nz[i].longitude, Latitude: nz[i].latitude, Name: nz[i].name, Bearing: b, Distance: d})

	}

	return l, nil
}

// Compass converts bearing (0-360) to a compass bearing name e.g., south-east.
func Compass(bearing float64) string {
	switch {
	case bearing >= 337.5 && bearing <= 360:
		return "north"
	case bearing >= 0 && bearing <= 22.5:
		return "north"
	case bearing > 22.5 && bearing < 67.5:
		return "north-east"
	case bearing >= 67.5 && bearing <= 112.5:
		return "east"
	case bearing > 112.5 && bearing < 157.5:
		return "south-east"
	case bearing >= 157.5 && bearing <= 202.5:
		return "south"
	case bearing > 202.5 && bearing < 247.5:
		return "south-west"
	case bearing >= 247.5 && bearing <= 292.5:
		return "west"
	case bearing > 292.5 && bearing < 337.5:
		return "north-west"
	default:
		return "north"
	}
}

// CompassShort converts bearing (0-360) to a short compass bearing name e.g., N.
func CompassShort(bearing float64) string {
	switch {
	case bearing >= 337.5 && bearing <= 360:
		return "N"
	case bearing >= 0 && bearing <= 22.5:
		return "N"
	case bearing > 22.5 && bearing < 67.5:
		return "NE"
	case bearing >= 67.5 && bearing <= 112.5:
		return "E"
	case bearing > 112.5 && bearing < 157.5:
		return "SE"
	case bearing >= 157.5 && bearing <= 202.5:
		return "S"
	case bearing > 202.5 && bearing < 247.5:
		return "SW"
	case bearing >= 247.5 && bearing <= 292.5:
		return "W"
	case bearing > 292.5 && bearing < 337.5:
		return "NW"
	default:
		return "N"
	}
}

func (l Locality) Description() string {
	if l.Distance < 5 {
		return "Within 5 km of " + l.Name
	}

	return fmt.Sprintf("%.f km %s of %s", math.Floor(l.Distance/5.0)*5, Compass(l.Bearing), l.Name)
}

func (l Locality) DescriptionShort() string {
	if l.Distance < 5 {
		return "Within 5 km of " + l.Name
	}

	return fmt.Sprintf("%.f km %s of %s", math.Floor(l.Distance/5.0)*5, CompassShort(l.Bearing), l.Name)
}
