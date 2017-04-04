package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	startTime, endTime Time
	mockFileData       string = "some fake test data"
)

func init() {
	tmp, err := time.Parse(
		time.RFC3339,
		"2012-11-01T22:08:41+00:00")
	if err != nil {
		panic("error parsing time")
	}

	startTime = Time{tmp}
	endTime = Time{startTime.Add(time.Hour * 1 * 24 * 312)}
}

// See https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3iface/ for s3iface docs and examples
type mockS3Client struct {
	s3iface.S3API
}

func newMockS3Client(p client.ConfigProvider, cfgs ...*aws.Config) s3iface.S3API {
	mc := mockS3Client{}
	return &mc
}

// mocked out ListObjectsV2 so we can give a known file list without connecting to S3
func (m *mockS3Client) ListObjectsV2(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	fileNames := []string{
		"NZ.CHST.01.LOG.D.2013.251",
		"NZ.CHST.01.LOG.D.2013.252",
		"NZ.CHST.01.LOG.D.2013.253",
		"NZ.ABCD.01.LOG.D.2013.251",
		"NZ.ABCD.01.LOG.D.2013.252",
		"NZ.ABCD.01.LOG.D.2013.253",
	}
	contents := []*s3.Object{
		{Key: &fileNames[0]},
		{Key: &fileNames[1]},
		{Key: &fileNames[2]},
		{Key: &fileNames[3]},
		{Key: &fileNames[4]},
		{Key: &fileNames[5]},
	}
	return &s3.ListObjectsV2Output{Contents: contents}, nil
}

func (c *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	// Only implementing what we'll parse, the output.Body which needs to be a ReadCloser
	rc := ioutil.NopCloser(strings.NewReader(mockFileData))
	out := s3.GetObjectOutput{Body: rc}
	return &out, nil
}

func TestMatchingFiles(t *testing.T) {
	bucket := "FAKEBUCKET"

	testCases := []struct {
		p            fdsnDataselectV1
		expectedKeys []string
		expectedData []byte
	}{
		// Normally we'd use newS3DataSource but we want to construct our own mock s3 client so create a new value instead
		{fdsnDataselectV1{
			StartTime: endTime,
			EndTime:   endTime,
			Network:   []string{"NZ"},
			Station:   []string{"CHST", "ABCD"},
			Location:  []string{"01"},
			Channel:   []string{"LOG"},
		},
			[]string{"NZ.CHST.01.LOG.D.2013.252", "NZ.ABCD.01.LOG.D.2013.252"},
			[]byte(mockFileData),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.p), func(t *testing.T) {
			ds := s3DataSource{bucket: bucket, params: tc.p, s3ClientFunc: newMockS3Client}

			keys, err := ds.matchingKeys()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(keys, tc.expectedKeys) {
				t.Errorf("Expected string slice %v but observed %v\n", tc.expectedKeys, keys)
			}

			for _, key := range tc.expectedKeys {
				data, err := ds.getObject(key)
				if err != nil {
					t.Error(err)
				}

				if !bytes.Equal(data, tc.expectedData) {
					t.Error("data different than expected")
				}
			}
		})
	}
}

func TestCommonString(t *testing.T) {
	testCases := []struct {
		input1, input2, wildcard, expected string
	}{
		{"aaaa", "aaab", "*", "aaa*"},
		{"1234", "5678", "?", "????"},
		{"2016", "2017", "?", "201?"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s, %s, %s", tc.input1, tc.expected, tc.wildcard), func(t *testing.T) {
			observed := commonString(tc.input1, tc.input2, tc.wildcard)
			if observed != tc.expected {
				t.Errorf("Expected string %s but observed %s", tc.expected, observed)
			}
		})
	}
}

func TestCommonSlice(t *testing.T) {
	// get the common prefix between a slice of input strings, all other chars being set to wildcard.
	testCases := []struct {
		inputs             []string
		wildcard, expected string
	}{
		{[]string{"aaaa", "aaab"}, "*", "aaa*"},
		{[]string{"aaaa", "aaab", "xyab"}, "*", "**a*"},
		{[]string{"1234", "5678"}, "?", "????"},
		{[]string{"2016", "2017"}, "?", "201?"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s, %s, %s", tc.inputs, tc.expected, tc.wildcard), func(t *testing.T) {
			observed := commonSlice(tc.inputs, tc.wildcard)
			if observed != tc.expected {
				t.Errorf("Expected string %s but observed %s", tc.expected, observed)
			}
		})
	}
}

func TestRegexp(t *testing.T) {
	testCases := []struct {
		inputParams     s3DataSource
		expectedPattern [][]string
		expectedRegexp  []string
		expectedPrefix  string
	}{
		{s3DataSource{
			params: fdsnDataselectV1{
				StartTime: startTime,
				EndTime:   startTime,
				Network:   []string{"NZ"},
				Station:   []string{"ABC"},
				Location:  []string{"XYZ"},
				Channel:   []string{"01"},
			},
		},
			[][]string{{"NZ"}, {"ABC"}, {"XYZ"}, {"01"}, {"D"}, {"2012"}, {"306"}},
			[]string{"(NZ)", "(ABC)", "(XYZ)", "(01)", "(D)", "(2012)", "(306)"},
			"NZ.ABC.XYZ.01.D.2012.306",
		},

		// using two different times, should get a different year and yearday
		{s3DataSource{
			params: fdsnDataselectV1{
				StartTime: endTime,
				EndTime:   startTime,
				Network:   []string{"NZ"},
				Station:   []string{"AB*"},
				Location:  []string{"?YZ"},
				Channel:   []string{"0*"},
			},
		},
			[][]string{{`NZ`}, {`AB*`}, {`?YZ`}, {`0*`}, {`D`}, {`201?`}, {`*`}},
			[]string{`(NZ)`, `(AB\w*)`, `(\w{1}YZ)`, `(0\w*)`, `(D)`, `(201\w{1})`, `(\w*)`},
			"NZ.AB",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.inputParams), func(t *testing.T) {
			observedPattern := tc.inputParams.searchPattern()
			if !reflect.DeepEqual(observedPattern, tc.expectedPattern) {
				t.Errorf("Expected string slice %v but observed %v", tc.expectedPattern, observedPattern)
			}

			observedRegexp := tc.inputParams.regexp()
			if !reflect.DeepEqual(observedRegexp, tc.expectedRegexp) {
				t.Errorf("Expected string %v but observed %v", tc.expectedRegexp, observedRegexp)
			}

			// the prefix used when querying files on S3 must stop at any wildcards in the key name
			observedPrefix := tc.inputParams.prefix()
			if observedPrefix != tc.expectedPrefix {
				t.Errorf("Expected string %s but observed %s", tc.expectedPrefix, observedPrefix)
			}
		})
	}
}
