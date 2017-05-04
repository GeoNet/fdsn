package msg

import (
	"fmt"
)

var (
	uaDepth     = []float64{15, 40, 100, 200}
	uaIntensity = []string{
		`unnoticeable`,
		`unnoticeable`,
		`unnoticeable`,
		`weak`,     // mmi 3
		`light`,    // mmi 4
		`moderate`, // mmi 5
		`strong`,   // mmi 6
		`severe`,   // mmi 7
	}
	uaLocalities = []Locality{
		// nz localities
		{Name: `ashburton`, Longitude: 171.75, Latitude: -43.9},
		{Name: `auckland`, Longitude: 174.77, Latitude: -36.85},
		{Name: `blenheim`, Longitude: 173.95, Latitude: -41.52},
		{Name: `cambridge`, Longitude: 175.47, Latitude: -37.88},
		{Name: `christchurch`, Longitude: 172.63, Latitude: -43.53},
		{Name: `dunedin`, Longitude: 170.5, Latitude: -45.88},
		{Name: `feilding`, Longitude: 175.57, Latitude: -40.23},
		{Name: `gisborne`, Longitude: 178.02, Latitude: -38.67},
		{Name: `gore`, Longitude: 168.93, Latitude: -46.1},
		{Name: `greymouth`, Longitude: 171.2, Latitude: -42.45},
		{Name: `hamilton`, Longitude: 175.28, Latitude: -37.78},
		{Name: `hastings`, Longitude: 176.85, Latitude: -39.65},
		{Name: `hawera`, Longitude: 174.28, Latitude: -39.58},
		{Name: `invercargill`, Longitude: 168.37, Latitude: -46.42},
		{Name: `levin`, Longitude: 175.28, Latitude: -40.62},
		{Name: `masterton`, Longitude: 175.65, Latitude: -40.95},
		{Name: `napier`, Longitude: 176.9, Latitude: -39.5},
		{Name: `nelson`, Longitude: 173.28, Latitude: -41.27},
		{Name: `new_plymouth`, Longitude: 174.07, Latitude: -39.07},
		{Name: `oamaru`, Longitude: 170.97, Latitude: -45.1},
		{Name: `palmerston_north`, Longitude: 175.62, Latitude: -40.37},
		{Name: `paraparaumu`, Longitude: 175.0, Latitude: -40.92},
		{Name: `pukekohe`, Longitude: 174.9, Latitude: -37.2},
		{Name: `queenstown`, Longitude: 168.67, Latitude: -45.03},
		{Name: `rotorua`, Longitude: 176.23, Latitude: -38.13},
		{Name: `taupo`, Longitude: 176.08, Latitude: -38.7},
		{Name: `tauranga`, Longitude: 176.17, Latitude: -37.68},
		{Name: `te_awamutu`, Longitude: 175.33, Latitude: -38.02},
		{Name: `timaru`, Longitude: 171.25, Latitude: -44.4},
		{Name: `tokoroa`, Longitude: 175.87, Latitude: -38.23},
		{Name: `whanganui`, Longitude: 175.05, Latitude: -39.93},
		{Name: `wellington`, Longitude: 174.77, Latitude: -41.28},
		{Name: `whakatane`, Longitude: 176.98, Latitude: -37.97},
		{Name: `whangarei`, Longitude: 174.32, Latitude: -35.72},
		// grid locations
		{Name: `167.5e47.5s`, Longitude: 167.5, Latitude: -47.5},
		{Name: `168.0e47.5s`, Longitude: 168.0, Latitude: -47.5},
		{Name: `167.0e47.0s`, Longitude: 167.0, Latitude: -47.0},
		{Name: `167.5e47.0s`, Longitude: 167.5, Latitude: -47.0},
		{Name: `168.0e47.0s`, Longitude: 168.0, Latitude: -47.0},
		{Name: `168.5e47.0s`, Longitude: 168.5, Latitude: -47.0},
		{Name: `169.0e47.0s`, Longitude: 169.0, Latitude: -47.0},
		{Name: `166.5e46.5s`, Longitude: 166.5, Latitude: -46.5},
		{Name: `167.0e46.5s`, Longitude: 167.0, Latitude: -46.5},
		{Name: `167.5e46.5s`, Longitude: 167.5, Latitude: -46.5},
		{Name: `168.0e46.5s`, Longitude: 168.0, Latitude: -46.5},
		{Name: `168.5e46.5s`, Longitude: 168.5, Latitude: -46.5},
		{Name: `169.0e46.5s`, Longitude: 169.0, Latitude: -46.5},
		{Name: `169.5e46.5s`, Longitude: 169.5, Latitude: -46.5},
		{Name: `170.0e46.5s`, Longitude: 170.0, Latitude: -46.5},
		{Name: `166.0e46.0s`, Longitude: 166.0, Latitude: -46.0},
		{Name: `166.5e46.0s`, Longitude: 166.5, Latitude: -46.0},
		{Name: `167.0e46.0s`, Longitude: 167.0, Latitude: -46.0},
		{Name: `167.5e46.0s`, Longitude: 167.5, Latitude: -46.0},
		{Name: `168.0e46.0s`, Longitude: 168.0, Latitude: -46.0},
		{Name: `168.5e46.0s`, Longitude: 168.5, Latitude: -46.0},
		{Name: `169.0e46.0s`, Longitude: 169.0, Latitude: -46.0},
		{Name: `169.5e46.0s`, Longitude: 169.5, Latitude: -46.0},
		{Name: `170.0e46.0s`, Longitude: 170.0, Latitude: -46.0},
		{Name: `170.5e46.0s`, Longitude: 170.5, Latitude: -46.0},
		{Name: `171.0e46.0s`, Longitude: 171.0, Latitude: -46.0},
		{Name: `166.0e45.5s`, Longitude: 166.0, Latitude: -45.5},
		{Name: `166.5e45.5s`, Longitude: 166.5, Latitude: -45.5},
		{Name: `167.0e45.5s`, Longitude: 167.0, Latitude: -45.5},
		{Name: `167.5e45.5s`, Longitude: 167.5, Latitude: -45.5},
		{Name: `168.0e45.5s`, Longitude: 168.0, Latitude: -45.5},
		{Name: `168.5e45.5s`, Longitude: 168.5, Latitude: -45.5},
		{Name: `169.0e45.5s`, Longitude: 169.0, Latitude: -45.5},
		{Name: `169.5e45.5s`, Longitude: 169.5, Latitude: -45.5},
		{Name: `170.0e45.5s`, Longitude: 170.0, Latitude: -45.5},
		{Name: `170.5e45.5s`, Longitude: 170.5, Latitude: -45.5},
		{Name: `171.0e45.5s`, Longitude: 171.0, Latitude: -45.5},
		{Name: `166.5e45.0s`, Longitude: 166.5, Latitude: -45.0},
		{Name: `167.0e45.0s`, Longitude: 167.0, Latitude: -45.0},
		{Name: `167.5e45.0s`, Longitude: 167.5, Latitude: -45.0},
		{Name: `168.0e45.0s`, Longitude: 168.0, Latitude: -45.0},
		{Name: `168.5e45.0s`, Longitude: 168.5, Latitude: -45.0},
		{Name: `169.0e45.0s`, Longitude: 169.0, Latitude: -45.0},
		{Name: `169.5e45.0s`, Longitude: 169.5, Latitude: -45.0},
		{Name: `170.0e45.0s`, Longitude: 170.0, Latitude: -45.0},
		{Name: `170.5e45.0s`, Longitude: 170.5, Latitude: -45.0},
		{Name: `171.0e45.0s`, Longitude: 171.0, Latitude: -45.0},
		{Name: `171.5e45.0s`, Longitude: 171.5, Latitude: -45.0},
		{Name: `167.0e44.5s`, Longitude: 167.0, Latitude: -44.5},
		{Name: `167.5e44.5s`, Longitude: 167.5, Latitude: -44.5},
		{Name: `168.0e44.5s`, Longitude: 168.0, Latitude: -44.5},
		{Name: `168.5e44.5s`, Longitude: 168.5, Latitude: -44.5},
		{Name: `169.0e44.5s`, Longitude: 169.0, Latitude: -44.5},
		{Name: `169.5e44.5s`, Longitude: 169.5, Latitude: -44.5},
		{Name: `170.0e44.5s`, Longitude: 170.0, Latitude: -44.5},
		{Name: `170.5e44.5s`, Longitude: 170.5, Latitude: -44.5},
		{Name: `171.0e44.5s`, Longitude: 171.0, Latitude: -44.5},
		{Name: `171.5e44.5s`, Longitude: 171.5, Latitude: -44.5},
		{Name: `172.0e44.5s`, Longitude: 172.0, Latitude: -44.5},
		{Name: `167.5e44.0s`, Longitude: 167.5, Latitude: -44.0},
		{Name: `168.0e44.0s`, Longitude: 168.0, Latitude: -44.0},
		{Name: `168.5e44.0s`, Longitude: 168.5, Latitude: -44.0},
		{Name: `169.0e44.0s`, Longitude: 169.0, Latitude: -44.0},
		{Name: `169.5e44.0s`, Longitude: 169.5, Latitude: -44.0},
		{Name: `170.0e44.0s`, Longitude: 170.0, Latitude: -44.0},
		{Name: `170.5e44.0s`, Longitude: 170.5, Latitude: -44.0},
		{Name: `171.0e44.0s`, Longitude: 171.0, Latitude: -44.0},
		{Name: `171.5e44.0s`, Longitude: 171.5, Latitude: -44.0},
		{Name: `172.0e44.0s`, Longitude: 172.0, Latitude: -44.0},
		{Name: `172.5e44.0s`, Longitude: 172.5, Latitude: -44.0},
		{Name: `173.0e44.0s`, Longitude: 173.0, Latitude: -44.0},
		{Name: `173.5e44.0s`, Longitude: 173.5, Latitude: -44.0},
		{Name: `168.5e43.5s`, Longitude: 168.5, Latitude: -43.5},
		{Name: `169.0e43.5s`, Longitude: 169.0, Latitude: -43.5},
		{Name: `169.5e43.5s`, Longitude: 169.5, Latitude: -43.5},
		{Name: `170.0e43.5s`, Longitude: 170.0, Latitude: -43.5},
		{Name: `170.5e43.5s`, Longitude: 170.5, Latitude: -43.5},
		{Name: `171.0e43.5s`, Longitude: 171.0, Latitude: -43.5},
		{Name: `171.5e43.5s`, Longitude: 171.5, Latitude: -43.5},
		{Name: `172.0e43.5s`, Longitude: 172.0, Latitude: -43.5},
		{Name: `172.5e43.5s`, Longitude: 172.5, Latitude: -43.5},
		{Name: `173.0e43.5s`, Longitude: 173.0, Latitude: -43.5},
		{Name: `173.5e43.5s`, Longitude: 173.5, Latitude: -43.5},
		{Name: `169.5e43.0s`, Longitude: 169.5, Latitude: -43.0},
		{Name: `170.0e43.0s`, Longitude: 170.0, Latitude: -43.0},
		{Name: `170.5e43.0s`, Longitude: 170.5, Latitude: -43.0},
		{Name: `171.0e43.0s`, Longitude: 171.0, Latitude: -43.0},
		{Name: `171.5e43.0s`, Longitude: 171.5, Latitude: -43.0},
		{Name: `172.0e43.0s`, Longitude: 172.0, Latitude: -43.0},
		{Name: `172.5e43.0s`, Longitude: 172.5, Latitude: -43.0},
		{Name: `173.0e43.0s`, Longitude: 173.0, Latitude: -43.0},
		{Name: `173.5e43.0s`, Longitude: 173.5, Latitude: -43.0},
		{Name: `174.0e43.0s`, Longitude: 174.0, Latitude: -43.0},
		{Name: `170.5e42.5s`, Longitude: 170.5, Latitude: -42.5},
		{Name: `171.0e42.5s`, Longitude: 171.0, Latitude: -42.5},
		{Name: `171.5e42.5s`, Longitude: 171.5, Latitude: -42.5},
		{Name: `172.0e42.5s`, Longitude: 172.0, Latitude: -42.5},
		{Name: `172.5e42.5s`, Longitude: 172.5, Latitude: -42.5},
		{Name: `173.0e42.5s`, Longitude: 173.0, Latitude: -42.5},
		{Name: `173.5e42.5s`, Longitude: 173.5, Latitude: -42.5},
		{Name: `174.0e42.5s`, Longitude: 174.0, Latitude: -42.5},
		{Name: `171.0e42.0s`, Longitude: 171.0, Latitude: -42.0},
		{Name: `171.5e42.0s`, Longitude: 171.5, Latitude: -42.0},
		{Name: `172.0e42.0s`, Longitude: 172.0, Latitude: -42.0},
		{Name: `172.5e42.0s`, Longitude: 172.5, Latitude: -42.0},
		{Name: `173.0e42.0s`, Longitude: 173.0, Latitude: -42.0},
		{Name: `173.5e42.0s`, Longitude: 173.5, Latitude: -42.0},
		{Name: `174.0e42.0s`, Longitude: 174.0, Latitude: -42.0},
		{Name: `174.5e42.0s`, Longitude: 174.5, Latitude: -42.0},
		{Name: `175.0e42.0s`, Longitude: 175.0, Latitude: -42.0},
		{Name: `175.5e42.0s`, Longitude: 175.5, Latitude: -42.0},
		{Name: `171.5e41.5s`, Longitude: 171.5, Latitude: -41.5},
		{Name: `172.0e41.5s`, Longitude: 172.0, Latitude: -41.5},
		{Name: `172.5e41.5s`, Longitude: 172.5, Latitude: -41.5},
		{Name: `173.0e41.5s`, Longitude: 173.0, Latitude: -41.5},
		{Name: `173.5e41.5s`, Longitude: 173.5, Latitude: -41.5},
		{Name: `174.0e41.5s`, Longitude: 174.0, Latitude: -41.5},
		{Name: `174.5e41.5s`, Longitude: 174.5, Latitude: -41.5},
		{Name: `175.0e41.5s`, Longitude: 175.0, Latitude: -41.5},
		{Name: `175.5e41.5s`, Longitude: 175.5, Latitude: -41.5},
		{Name: `176.0e41.5s`, Longitude: 176.0, Latitude: -41.5},
		{Name: `171.5e41.0s`, Longitude: 171.5, Latitude: -41.0},
		{Name: `172.0e41.0s`, Longitude: 172.0, Latitude: -41.0},
		{Name: `172.5e41.0s`, Longitude: 172.5, Latitude: -41.0},
		{Name: `173.0e41.0s`, Longitude: 173.0, Latitude: -41.0},
		{Name: `173.5e41.0s`, Longitude: 173.5, Latitude: -41.0},
		{Name: `174.0e41.0s`, Longitude: 174.0, Latitude: -41.0},
		{Name: `174.5e41.0s`, Longitude: 174.5, Latitude: -41.0},
		{Name: `175.0e41.0s`, Longitude: 175.0, Latitude: -41.0},
		{Name: `175.5e41.0s`, Longitude: 175.5, Latitude: -41.0},
		{Name: `176.0e41.0s`, Longitude: 176.0, Latitude: -41.0},
		{Name: `176.5e41.0s`, Longitude: 176.5, Latitude: -41.0},
		{Name: `172.0e40.5s`, Longitude: 172.0, Latitude: -40.5},
		{Name: `172.5e40.5s`, Longitude: 172.5, Latitude: -40.5},
		{Name: `173.0e40.5s`, Longitude: 173.0, Latitude: -40.5},
		{Name: `173.5e40.5s`, Longitude: 173.5, Latitude: -40.5},
		{Name: `174.0e40.5s`, Longitude: 174.0, Latitude: -40.5},
		{Name: `174.5e40.5s`, Longitude: 174.5, Latitude: -40.5},
		{Name: `175.0e40.5s`, Longitude: 175.0, Latitude: -40.5},
		{Name: `175.5e40.5s`, Longitude: 175.5, Latitude: -40.5},
		{Name: `176.0e40.5s`, Longitude: 176.0, Latitude: -40.5},
		{Name: `176.5e40.5s`, Longitude: 176.5, Latitude: -40.5},
		{Name: `177.0e40.5s`, Longitude: 177.0, Latitude: -40.5},
		{Name: `174.0e40.0s`, Longitude: 174.0, Latitude: -40.0},
		{Name: `174.5e40.0s`, Longitude: 174.5, Latitude: -40.0},
		{Name: `175.0e40.0s`, Longitude: 175.0, Latitude: -40.0},
		{Name: `175.5e40.0s`, Longitude: 175.5, Latitude: -40.0},
		{Name: `176.0e40.0s`, Longitude: 176.0, Latitude: -40.0},
		{Name: `176.5e40.0s`, Longitude: 176.5, Latitude: -40.0},
		{Name: `177.0e40.0s`, Longitude: 177.0, Latitude: -40.0},
		{Name: `177.5e40.0s`, Longitude: 177.5, Latitude: -40.0},
		{Name: `173.5e39.5s`, Longitude: 173.5, Latitude: -39.5},
		{Name: `174.0e39.5s`, Longitude: 174.0, Latitude: -39.5},
		{Name: `174.5e39.5s`, Longitude: 174.5, Latitude: -39.5},
		{Name: `175.0e39.5s`, Longitude: 175.0, Latitude: -39.5},
		{Name: `175.5e39.5s`, Longitude: 175.5, Latitude: -39.5},
		{Name: `176.0e39.5s`, Longitude: 176.0, Latitude: -39.5},
		{Name: `176.5e39.5s`, Longitude: 176.5, Latitude: -39.5},
		{Name: `177.0e39.5s`, Longitude: 177.0, Latitude: -39.5},
		{Name: `177.5e39.5s`, Longitude: 177.5, Latitude: -39.5},
		{Name: `178.0e39.5s`, Longitude: 178.0, Latitude: -39.5},
		{Name: `178.5e39.5s`, Longitude: 178.5, Latitude: -39.5},
		{Name: `173.5e39.0s`, Longitude: 173.5, Latitude: -39.0},
		{Name: `174.0e39.0s`, Longitude: 174.0, Latitude: -39.0},
		{Name: `174.5e39.0s`, Longitude: 174.5, Latitude: -39.0},
		{Name: `175.0e39.0s`, Longitude: 175.0, Latitude: -39.0},
		{Name: `175.5e39.0s`, Longitude: 175.5, Latitude: -39.0},
		{Name: `176.0e39.0s`, Longitude: 176.0, Latitude: -39.0},
		{Name: `176.5e39.0s`, Longitude: 176.5, Latitude: -39.0},
		{Name: `177.0e39.0s`, Longitude: 177.0, Latitude: -39.0},
		{Name: `177.5e39.0s`, Longitude: 177.5, Latitude: -39.0},
		{Name: `178.0e39.0s`, Longitude: 178.0, Latitude: -39.0},
		{Name: `178.5e39.0s`, Longitude: 178.5, Latitude: -39.0},
		{Name: `174.0e38.5s`, Longitude: 174.0, Latitude: -38.5},
		{Name: `174.5e38.5s`, Longitude: 174.5, Latitude: -38.5},
		{Name: `175.0e38.5s`, Longitude: 175.0, Latitude: -38.5},
		{Name: `175.5e38.5s`, Longitude: 175.5, Latitude: -38.5},
		{Name: `176.0e38.5s`, Longitude: 176.0, Latitude: -38.5},
		{Name: `176.5e38.5s`, Longitude: 176.5, Latitude: -38.5},
		{Name: `177.0e38.5s`, Longitude: 177.0, Latitude: -38.5},
		{Name: `177.5e38.5s`, Longitude: 177.5, Latitude: -38.5},
		{Name: `178.0e38.5s`, Longitude: 178.0, Latitude: -38.5},
		{Name: `178.5e38.5s`, Longitude: 178.5, Latitude: -38.5},
		{Name: `174.5e38.0s`, Longitude: 174.5, Latitude: -38.0},
		{Name: `175.0e38.0s`, Longitude: 175.0, Latitude: -38.0},
		{Name: `175.5e38.0s`, Longitude: 175.5, Latitude: -38.0},
		{Name: `176.0e38.0s`, Longitude: 176.0, Latitude: -38.0},
		{Name: `176.5e38.0s`, Longitude: 176.5, Latitude: -38.0},
		{Name: `177.0e38.0s`, Longitude: 177.0, Latitude: -38.0},
		{Name: `177.5e38.0s`, Longitude: 177.5, Latitude: -38.0},
		{Name: `178.0e38.0s`, Longitude: 178.0, Latitude: -38.0},
		{Name: `178.5e38.0s`, Longitude: 178.5, Latitude: -38.0},
		{Name: `174.5e37.5s`, Longitude: 174.5, Latitude: -37.5},
		{Name: `175.0e37.5s`, Longitude: 175.0, Latitude: -37.5},
		{Name: `175.5e37.5s`, Longitude: 175.5, Latitude: -37.5},
		{Name: `176.0e37.5s`, Longitude: 176.0, Latitude: -37.5},
		{Name: `176.5e37.5s`, Longitude: 176.5, Latitude: -37.5},
		{Name: `177.0e37.5s`, Longitude: 177.0, Latitude: -37.5},
		{Name: `177.5e37.5s`, Longitude: 177.5, Latitude: -37.5},
		{Name: `178.0e37.5s`, Longitude: 178.0, Latitude: -37.5},
		{Name: `178.5e37.5s`, Longitude: 178.5, Latitude: -37.5},
		{Name: `174.0e37.0s`, Longitude: 174.0, Latitude: -37.0},
		{Name: `174.5e37.0s`, Longitude: 174.5, Latitude: -37.0},
		{Name: `175.0e37.0s`, Longitude: 175.0, Latitude: -37.0},
		{Name: `175.5e37.0s`, Longitude: 175.5, Latitude: -37.0},
		{Name: `176.0e37.0s`, Longitude: 176.0, Latitude: -37.0},
		{Name: `176.5e37.0s`, Longitude: 176.5, Latitude: -37.0},
		{Name: `174.0e36.5s`, Longitude: 174.0, Latitude: -36.5},
		{Name: `174.5e36.5s`, Longitude: 174.5, Latitude: -36.5},
		{Name: `175.0e36.5s`, Longitude: 175.0, Latitude: -36.5},
		{Name: `175.5e36.5s`, Longitude: 175.5, Latitude: -36.5},
		{Name: `176.0e36.5s`, Longitude: 176.0, Latitude: -36.5},
		{Name: `173.5e36.0s`, Longitude: 173.5, Latitude: -36.0},
		{Name: `174.0e36.0s`, Longitude: 174.0, Latitude: -36.0},
		{Name: `174.5e36.0s`, Longitude: 174.5, Latitude: -36.0},
		{Name: `175.0e36.0s`, Longitude: 175.0, Latitude: -36.0},
		{Name: `175.5e36.0s`, Longitude: 175.5, Latitude: -36.0},
		{Name: `176.0e36.0s`, Longitude: 176.0, Latitude: -36.0},
		{Name: `173.0e35.5s`, Longitude: 173.0, Latitude: -35.5},
		{Name: `173.5e35.5s`, Longitude: 173.5, Latitude: -35.5},
		{Name: `174.0e35.5s`, Longitude: 174.0, Latitude: -35.5},
		{Name: `174.5e35.5s`, Longitude: 174.5, Latitude: -35.5},
		{Name: `175.0e35.5s`, Longitude: 175.0, Latitude: -35.5},
		{Name: `173.0e35.0s`, Longitude: 173.0, Latitude: -35.0},
		{Name: `173.5e35.0s`, Longitude: 173.5, Latitude: -35.0},
		{Name: `174.0e35.0s`, Longitude: 174.0, Latitude: -35.0},
		{Name: `174.5e35.0s`, Longitude: 174.5, Latitude: -35.0},
		{Name: `172.5e34.5s`, Longitude: 172.5, Latitude: -34.5},
		{Name: `173.0e34.5s`, Longitude: 173.0, Latitude: -34.5},
		{Name: `173.5e34.5s`, Longitude: 173.5, Latitude: -34.5},
		{Name: `173.0e34.0s`, Longitude: 173.0, Latitude: -34.0},
	}
)

// uaTags generates tags for sending to Urban Airship.
func (q *Quake) uaTags() (tags []string) {
	if q.err != nil {
		return
	}

	// magnitude and depth tags
	for i := 3; i <= int(q.Magnitude); i++ {
		for _, d := range uaDepth {
			if q.Depth < d {
				tags = append(tags, fmt.Sprintf("mag%d_depth%.0f", i, d))
			}
		}
		// the 'any' depth tag
		tags = append(tags, fmt.Sprintf("mag%d_depth", i))
	}

	// intensity tags
	mmi := q.MMI()

	if mmi < 3.0 {
		return
	}

	mmiIndex := int(mmi) + 1
	if mmiIndex > len(uaIntensity) {
		mmiIndex = len(uaIntensity)
	}

	intensities := uaIntensity[3:mmiIndex]
	for _, i := range intensities {
		tags = append(tags, fmt.Sprintf("%s@all_locations", i))
	}

	// intensity at locality or grid point
	for _, l := range uaLocalities {
		d, _ := geo.To(l.Latitude, l.Longitude, q.Latitude, q.Longitude)

		mmiD := MMIDistance(d, q.Depth, mmi)

		if mmiD >= 3.0 {
			mmiIndex := int(mmiD) + 1
			if mmiIndex > len(uaIntensity) {
				mmiIndex = len(uaIntensity)
			}

			intensities := uaIntensity[3:mmiIndex]

			for _, i := range intensities {
				tags = append(tags, fmt.Sprintf("%s@%s", i, l.Name))
			}
		}
	}

	return
}
