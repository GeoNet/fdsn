package internal

/*
 */
type ID int16

const (
	// HTTP requests
	Requests ID = 1

	// HTTP status codes (100 - 999).
	StatusOK                  ID = 200
	StatusBadRequest          ID = 400
	StatusUnauthorized        ID = 401
	StatusNotFound            ID = 404
	StatusInternalServerError ID = 500
	StatusServiceUnavailable  ID = 503

	// MemStats cf pkg/runtime/#MemStats
	// Also https://software.intel.com/en-us/blogs/2014/05/10/debugging-performance-issues-in-go-programs
	MemSys         ID = 1000 // bytes obtained from system
	MemHeapAlloc   ID = 1001 // bytes allocated and not yet freed
	MemHeapSys     ID = 1002 // bytes obtained from system
	MemHeapObjects ID = 1003 // total number of allocated objects

	// Other runtime stats
	Routines ID = 1100 // number of Go routines in use.

	// Messaging
	MsgRx   ID = 1201
	MsgTx   ID = 1202
	MsgProc ID = 1203
	MsgErr  ID = 1204

	// Timer
	AvgMean   ID = 2001
	MaxFifty  ID = 2002
	MaxNinety ID = 2003

	// Data latency
	Mean   ID = 3001
	Fifty  ID = 3002
	Ninety ID = 3003
)

var idColours = map[int]string{
	1:   "#4daf4a",
	200: "deepskyblue",
	400: "#984ea3",
	401: "#a65628",
	404: "#ff7f00",
	500: "#e41a1c",
	503: "#e41a1c",

	1000: "#a6cee3",
	1001: "#1f78b4",
	1002: "#b2df8a",
	1003: "deepskyblue",

	1100: "deepskyblue",

	1201: "#4daf4a",
	1202: "#984ea3",
	1203: "deepskyblue",
	1204: "#e41a1c",

	2001: "#ff0000",
	2002: "#00ff00",
	2003: "#0000ff",

	3001: "deepskyblue",
	3002: "deeppink",
	3003: "limegreen",
}

var idLabels = map[int]string{
	1:   "Requests",
	200: "200 OK",
	400: "400 Bad Request",
	401: "401 Unauthorized",
	404: "404 Not Found",
	500: "500 Internal Server Error",
	503: "503 Service Unavailable",

	1000: "Mem Sys",
	1001: "Mem Heap Alloc",
	1002: "Mem Heap Sys",
	1003: "Mem Heap Objects",

	1100: "Go Routines",

	1201: "Msg Rx",
	1202: "Msg Tx",
	1203: "Msg Processed",
	1204: "Msg Error",

	2001: "Avg Mean",
	2002: "Max Fifty",
	2003: "Max Ninety",

	3001: "Mean",
	3002: "Fifty",
	3003: "Ninety",
}

func Colour(id int) string {
	if s, ok := idColours[id]; ok {
		return s
	}

	return "yellow"
}

func Label(id int) string {
	if s, ok := idLabels[id]; ok {
		return s
	}

	return "que"
}
