// +build integration

package main

import (
	"os"
	"testing"
)

// TestHoldings runs an integration test against S3
// Env var need to be set (see env.list).
// Run using:
// go test -tags integration -v -run TestHoldingsS3
func TestHoldingsS3(t *testing.T) {
	h, err := holdingS3(os.Getenv("S3_BUCKET"), "NZ.ALRZ.10.EHZ.D.2017.023")
	if err != nil {
		t.Error(err)
	}

	t.Logf("%+v", h)
}
