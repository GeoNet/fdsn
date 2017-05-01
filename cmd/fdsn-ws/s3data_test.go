package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

var (
	startTime, endTime Time
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

func TestRegexp(t *testing.T) {
	testCases := []struct {
		inputParams    s3DataSource
		expectedRegexp []string
	}{
		{s3DataSource{
			params: fdsnDataselectV1{
				StartTime: startTime,
				EndTime:   endTime,
				Network:   []string{"NZ"},
				Station:   []string{"ABC"},
				Location:  []string{"XYZ"},
				Channel:   []string{"01"},
			},
		},
			[]string{"(^NZ$)", "(^ABC$)", "(^XYZ$)", "(^01$)"},
		},

		{s3DataSource{
			params: fdsnDataselectV1{
				StartTime: startTime,
				EndTime:   endTime,
				Network:   []string{"NZ"},
				Station:   []string{"AB*"},
				Location:  []string{"?YZ", "EF*"},
				Channel:   []string{"0*"},
			},
		},
			[]string{`(^NZ$)`, `(^AB\w*$)`, `(^\w{1}YZ$)|(^EF\w*$)`, `(^0\w*$)`},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.inputParams), func(t *testing.T) {
			net, sta, loc, cha, start, end := tc.inputParams.regexp()
			observedRegexp := []string{net, sta, loc, cha}
			if !reflect.DeepEqual(observedRegexp, tc.expectedRegexp) {
				t.Errorf("Expected string %v but observed %v", tc.expectedRegexp, observedRegexp)
			}

			if !start.Equal(startTime.Time) {
				t.Errorf("Expected time %v but observed %v", startTime, start)
			}

			if !end.Equal(endTime.Time) {
				t.Errorf("Expected time %v but observed %v", endTime, end)
			}
		})
	}
}
