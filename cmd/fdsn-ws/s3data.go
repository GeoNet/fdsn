package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func newS3Client(p client.ConfigProvider, cfgs ...*aws.Config) s3iface.S3API {
	return s3.New(p, cfgs...)
}

type s3DataSource struct {
	params       fdsnDataselectV1
	s3ClientFunc func(client.ConfigProvider, ...*aws.Config) s3iface.S3API
	bucket       string
	session      *session.Session
	cfg          *aws.Config
}

func newS3DataSource(bucket string, dsParams fdsnDataselectV1, maxRetries int) (*s3DataSource, error) {
	var err error
	var s3Session *session.Session

	cfg := aws.NewConfig()
	// either we specify maxRetries or a Retryer but no need for both, will use DefaultRetryer by default
	cfg.MaxRetries = &maxRetries
	// an interesting commit to fix timeouts/etc for strava: https://github.com/strava/goamz/commit/0013fca27c5c3b849d9b2d8c3c15f59a53ec38e9
	cfg.HTTPClient = http.DefaultClient
	cfg.HTTPClient.Timeout = HTTP_TIMEOUT

	if s3Session, err = session.NewSession(cfg); err != nil {
		return nil, err
	}

	s := s3DataSource{
		bucket:       bucket,
		params:       dsParams,
		s3ClientFunc: newS3Client, // Using a function so we can create new clients at will and mock with s3iface.
		session:      s3Session,
		cfg:          cfg,
	}

	return &s, nil
}

// getS3Object downloads the specified key from an S3 bucket and returns a byte slice containing the data.  This
// is thread safe and creates it's own S3 client.
func (s *s3DataSource) getObject(key string) (b []byte, err error) {
	getParams := s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}

	// create a new client for each file to download, reuse the session and config
	c := s.s3ClientFunc(s.session)
	var output *s3.GetObjectOutput
	if output, err = c.GetObject(&getParams); err != nil {
		return nil, err
	}
	defer output.Body.Close()

	if b, err = ioutil.ReadAll(output.Body); err != nil {
		return nil, err
	}

	return b, nil
}

// matchingKeys returns a slice of strings (S3 keys - filenames) that match the search parameters.
func (s *s3DataSource) matchingKeys() (keys []string, err error) {
	prefix := s.prefix()
	listParams := &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &prefix,
	}

	var resp *s3.ListObjectsV2Output
	c := s.s3ClientFunc(s.session)
	var contents []*s3.Object

	// poorly documented: if resp.isTruncated is true we need to keep reading in chunks of 1000 until it is false
	for {
		if resp, err = c.ListObjectsV2(listParams); err != nil {
			return nil, err
		}

		contents = append(contents, resp.Contents...)

		// AWS using pointers to bools (?!) so need to check for nil
		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		listParams.ContinuationToken = resp.NextContinuationToken
	}

	var re *regexp.Regexp
	if re, err = regexp.Compile(strings.Join(s.regexp(), `\.`)); err != nil {
		return nil, err
	}

	for _, value := range contents {
		if !re.MatchString(*value.Key) {
			continue
		}

		// The regexp is weak at parsing the day of year so do it manually here
		parts := strings.Split(*value.Key, ".")
		var yearDayInt int
		if yearDayInt, err = strconv.Atoi(parts[len(parts)-1]); err != nil {
			return nil, err
		}

		if yearDayInt < s.params.StartTime.YearDay() || yearDayInt > s.params.EndTime.YearDay() {
			continue
		}

		keys = append(keys, *value.Key)
	}

	return keys, nil
}

// prefix returns the prefix string to be used when querying all matching keys from an S3 bucket.  This cannot include
// a full regexp search pattern since AWS does not support this.
func (s *s3DataSource) prefix() (prefix string) {
	var merged []string
	for _, params := range s.searchPattern() {
		merged = append(merged, commonSlice(params, "*"))
	}

	// s3 prefix does not support wildcards, so truncate if they're present
	prefix = strings.Join(merged, ".")
	prefix = strings.Split(prefix, "*")[0]
	prefix = strings.Split(prefix, "?")[0]
	// location can have two spaces or "--", neither of which exist in the S3 key name
	prefix = strings.Split(prefix, " ")[0]
	prefix = strings.Split(prefix, "--")[0]

	return prefix
}

// regexp returns a regexp string to see if an S3 key matches the input parameters.  It converts
// the '*', '?', ' ' and '--' characters to their regular expression equivalents for pattern matching with Go's regexp.
func (s *s3DataSource) regexp() (pattern []string) {

	toPattern := func(params []string) (out string) {
		var newParams []string
		for _, param := range params {
			newParam := strings.Replace(param, "*", `\w*`, -1)
			newParam = strings.Replace(newParam, "?", `\w{1}`, -1)
			// blank or missing locations, we convert spaces and two dashes to wildcards for the regexp
			newParam = strings.Replace(newParam, "--", `\w{2}`, -1)
			newParam = strings.Replace(newParam, " ", `\w{1}`, -1)
			newParams = append(newParams, newParam)
		}

		return "(" + strings.Join(newParams, "|") + ")"
	}

	fields := s.searchPattern()
	for _, params := range fields {
		pattern = append(pattern, toPattern(params))
	}

	return pattern
}

func (s *s3DataSource) searchPattern() [][]string {
	startYear := fmt.Sprintf("%04d", s.params.StartTime.Year())
	endYear := fmt.Sprintf("%04d", s.params.EndTime.Year())
	year := commonString(startYear, endYear, "?")

	startDoy := fmt.Sprintf("%03d", s.params.StartTime.YearDay())
	endDoy := fmt.Sprintf("%03d", s.params.EndTime.YearDay())

	// if we're looking at multiple years then parse every day of year
	var doy string
	if year == startYear && year == endYear {
		doy = commonString(startDoy, endDoy, "?")
	} else {
		doy = "*"
	}

	return [][]string{s.params.Network, s.params.Station, s.params.Location, s.params.Channel, {"D"}, {year}, {doy}}
}

// commonSlice constructs a string from a slice of strings where any non-matching characters are replaced by the string wildCard
func commonSlice(strs []string, wildcard string) (output string) {

	if len(strs) == 0 {
		return ""
	}

	if len(strs) == 1 {
		return strs[0]
	}

	output = strs[0]
	for _, s2 := range strs[1:] {
		output = commonString(output, s2, wildcard)
	}

	return output
}

// commonString constructs a string from two input strings where any non-matching characters are replaced by the string wildCard
func commonString(s1, s2, wildcard string) (output string) {
	if s1 == s2 {
		output = s1
	} else {
		for i, c := range s1 {
			if c == rune(s2[i]) {
				output += string(c)
			} else {
				output += wildcard
			}
		}
	}

	return output
}
