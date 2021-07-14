package sl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const versionFinderString = `^SeedLink v(\d)\.(\d)`
const timeFormat = "2006,01,02,15,04,05"

var versionFinder = regexp.MustCompile(versionFinderString)

const (
	cmdHello = "HELLO"
	cmdCat   = "CAT" //Not implemented by Ringserver
	cmdClose = "BYE"

	cmdStation = "STATION" //Enables multi-station mode: STATION station code [network code]
	cmdEnd     = "END"     //End of handshaking for multi-station mode

	cmdSelect = "SELECT" //   SELECT [pattern]
	cmdData   = "DATA"   // DATA [n [begin time]]
	//cmdFetch  = "FETCH"  // FETCH [n [begin time]]
	cmdTime = "TIME" // TIME [begin time [end time]]

	cmdInfoId           = "INFO ID"
	cmdInfoCapabilities = "INFO CAPABILITIES"
	cmdInfoStations     = "INFO STATIONS"
	cmdInfoStreams      = "INFO STREAMS"
	cmdInfoGaps         = "INFO GAPS"
	cmdInfoConnections  = "INFO CONNECTIONS"
	cmdInfoAll          = "INFO ALL"

	cmdCrLf = "\r\n"
)

var infoLevel = map[string]struct {
	capability string
	command    string
}{
	"ID":           {"info:id", cmdInfoId},
	"CAPABILITIES": {"info:capabilities", cmdInfoCapabilities},
	"STATIONS":     {"info:stations", cmdInfoStations},
	"STREAMS":      {"info:streams", cmdInfoStreams},
	"GAPS":         {"info:gaps", cmdInfoGaps},
	"CONNECTIONS":  {"info:connections", cmdInfoConnections},
	"ALL":          {"info:all", cmdInfoAll},
}

const capabilityWildCard = "NSWILDCARD"

type Conn struct {
	net.Conn
	timeout time.Duration

	rawVersion string
	version    struct {
		major, minor int
	}

	capabilities map[string]bool
}

// NewConn returns a new connection to the named seedlink server with a given command timeout. It is expected that the
// Close function be called when the connection is no longer required.
func NewConn(service string, timeout time.Duration) (*Conn, error) {
	if !strings.Contains(service, ":") {
		service = net.JoinHostPort(service, "18000")
	}

	client, err := net.Dial("tcp", service)
	if err != nil {
		return nil, err
	}

	conn := Conn{
		Conn:    client,
		timeout: timeout,
	}

	if err := conn.getCapabilities(); err != nil {
		_ = conn.Close()

		return nil, err
	}

	return &conn, nil
}

func (c *Conn) setDeadline() error {
	if !(c.timeout > 0) {
		return nil
	}
	return c.SetDeadline(time.Now().Add(c.timeout))
}

func (c *Conn) readPacket() (*Packet, error) {

	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, c, PacketSize); err != nil {
		return nil, err
	}

	pkt, err := NewPacket(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return pkt, nil
}

func (c *Conn) writeString(str string) (int, error) {
	if err := c.setDeadline(); err != nil {
		return 0, err
	}
	return c.Write([]byte(str + cmdCrLf))
}

func (c *Conn) infoCommand(cmd string) ([]byte, error) {

	if _, err := c.writeString(cmd); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	for {
		pkt, err := c.readPacket()
		if err != nil {
			return nil, err
		}
		offset := binary.BigEndian.Uint16(pkt.Data[44:46])
		buf.WriteString(string(pkt.Data[offset:]))

		if pkt.Seq[5] != '*' {
			break
		}

	}

	return buf.Bytes(), nil
}

func (c *Conn) issueCommand(cmd string) ([]byte, error) {

	if _, err := c.writeString(cmd); err != nil {
		return nil, err
	}

	b := make([]byte, 512)

	i, err := c.Read(b)
	if err != nil {
		return nil, err
	}

	if s := string(b[:i]); strings.HasPrefix(s, "ERROR") {
		return nil, fmt.Errorf("got ERROR response: %v", s)
	}

	return b[:i], nil
}

func (c *Conn) modifierCommand(cmd string) error {

	if _, err := c.writeString(cmd); err != nil {
		return err
	}

	b := make([]byte, 10)

	i, err := c.Read(b)
	if err != nil {
		return err
	}

	if s := string(b[:i]); !strings.HasPrefix(s, "OK") {
		return fmt.Errorf("non-OK response from server: %v", strings.TrimSpace(s))
	}

	return nil
}

func (c *Conn) actionCommand(cmd string) error {

	if _, err := c.writeString(cmd); err != nil {
		return err
	}

	return nil
}

func parseSeedlinkVersion(hello string) (int, int) {
	match := versionFinder.FindStringSubmatch(hello)

	if len(match) == 0 {
		return 0, 0
	}

	major, _ := strconv.ParseInt(match[1], 10, 32)
	minor, _ := strconv.ParseInt(match[2], 10, 32)

	return int(major), int(minor)
}

func (c *Conn) getCapabilities() error {
	hello, err := c.issueCommand(cmdHello) // Use this to get some initial version/capability information.
	if err != nil {
		return fmt.Errorf("failed to issue a 'hello' command: %v", err)
	}

	c.rawVersion = string(hello)
	c.capabilities = make(map[string]bool)

	// h is like:
	// SeedLink v3.1 (2017.052 RingServer) :: SLPROTO:3.1 CAP EXTREPLY NSWILDCARD BATCH WS:13
	// GeoNet SeedLink Server
	// TODO: Can we implement EXTREPLY CAP reporting?
	// TODO: Investigate BATCH

	c.version.major, c.version.minor = parseSeedlinkVersion(string(hello))

	if caps := strings.Split(strings.Split(string(hello), cmdCrLf)[0], "::"); len(caps) == 2 {
		for _, hc := range strings.Split(caps[1], " ") {
			c.capabilities[hc] = true
		}
	}

	capinfo, err := c.infoCommand(cmdInfoCapabilities)
	if err != nil {
		return fmt.Errorf("unable to list capabilities: %v", err)
	}

	var info Info
	if err := info.Unmarshal(capinfo); err != nil {
		return fmt.Errorf("could not parse capabilities XML: %v", err)
	}

	for _, i := range info.Capability {
		c.capabilities[i.Name] = true
	}

	return nil
}

// GetInfoLevel requests the seedlink server return an INFO request for the given level.
func (c *Conn) GetInfoLevel(level string) ([]byte, error) {
	info, ok := infoLevel[strings.ToUpper(level)]
	if !ok {
		return nil, fmt.Errorf("unknown info level: %v", level)
	}
	if !c.capabilities[info.capability] {
		return nil, fmt.Errorf("capability %s not present", info.capability)
	}

	return c.infoCommand(info.command)
}

// GetInfo requests the seedlink server return an INFO request for the given level. The results
// are returned as a decoded Info pointer, or an error otherwise.
func (c *Conn) GetInfo(level string) (*Info, error) {
	data, err := c.GetInfoLevel(level)
	if err != nil {
		return nil, err
	}

	var info Info
	if err := info.Unmarshal(data); err != nil {
		return nil, err
	}

	return &info, nil
}

// CommandId sends an INFO ID command to the seedlink server.
func (c *Conn) CommandId() ([]byte, error) {
	return c.infoCommand(cmdInfoId)
}

// CommandHello sends a HELLO command to the seedlink server.
func (c *Conn) CommandHello() ([]byte, error) {
	return c.infoCommand(cmdHello)
}

// CommandClose sends a BYE command to the seedlink server.
func (c *Conn) CommandClose() ([]byte, error) {
	return c.infoCommand(cmdClose)
}

// CommandStationList sends a CAT command to the seedlink server.
func (c *Conn) CommandCat() ([]byte, error) {
	return c.infoCommand(cmdCat)
}

// CommandStation sends a STATION command to the seedlink server.
func (c *Conn) CommandStation(station, network string) error {
	if strings.ContainsAny(station, "*?") && !c.capabilities[capabilityWildCard] {
		return fmt.Errorf("station selector '%s' contains wildcards but the server does not report capability NSWILDCARD", station)
	}
	if strings.ContainsAny(network, "*?") && !c.capabilities[capabilityWildCard] {
		return fmt.Errorf("network selector '%s' contains wildcards but the server does not report capability NSWILDCARD", network)
	}
	switch {
	case network != "":
		if err := c.modifierCommand(fmt.Sprintf("%s %s %s", cmdStation, station, network)); err != nil {
			return fmt.Errorf("error sending STATION %s %s: %v", station, network, err)
		}
	default:
		if err := c.modifierCommand(fmt.Sprintf("%s %s", cmdStation, station)); err != nil {
			return fmt.Errorf("error sending STATION %s: %v", station, err)
		}
	}

	return nil
}

// CommandSelect sends a SELECT command to the seedlink server.
func (c *Conn) CommandSelect(selection string) error {

	if err := c.modifierCommand(fmt.Sprintf("%s %s", cmdSelect, selection)); err != nil {
		return fmt.Errorf("error sending SELECT %s: %v", selection, err)
	}

	return nil
}

// CommandData sends a DATA command to the seedlink server.
func (c *Conn) CommandData(sequence string, starttime time.Time) error {

	var dc string
	switch {
	case sequence == "":
		dc = cmdData
	case starttime.IsZero():
		dc = fmt.Sprintf("%s %s\n", cmdData, sequence)
	default:
		dc = fmt.Sprintf("%s %s %s\n", cmdData, sequence, starttime.Format(timeFormat))
	}

	if err := c.modifierCommand(dc); err != nil {
		return fmt.Errorf("error sending DATA: %v", err)
	}

	return nil
}

// CommandTime sends a TIME command to the seedlink server.
func (c *Conn) CommandTime(starttime, endtime time.Time) error {

	if starttime.IsZero() {
		return nil
	}

	var tc string
	switch {
	case endtime.IsZero():
		tc = fmt.Sprintf("%s %s\n", cmdTime, starttime.Format(timeFormat))
	default:
		tc = fmt.Sprintf("%s %s %s\n", cmdTime, starttime.Format(timeFormat), endtime.Format(timeFormat))
	}

	if err := c.modifierCommand(tc); err != nil {
		return fmt.Errorf("error sending TIME: %v", err)
	}

	return nil
}

// CommandEnd sends an END command to the seedlink server.
func (c *Conn) CommandEnd() error {
	if err := c.actionCommand(cmdEnd); err != nil {
		return fmt.Errorf("error sending END: %v", err)
	}
	return nil
}

// Collect returns a seedlink packet if available within the optional timout. Any error returned should be
// checked that it isn't a timeout, this should be handled as appropriate for the request.
func (c *Conn) Collect() (*Packet, error) {
	if err := c.setDeadline(); err != nil {
		return nil, err
	}
	return c.readPacket()
}
