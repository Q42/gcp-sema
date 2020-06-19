package main

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func TestSortPutsLatestVersionFirst(t *testing.T) {
	var versions = []*secretmanagerpb.SecretVersion{
		{CreateTime: &timestamp.Timestamp{Seconds: 42}},
		{CreateTime: &timestamp.Timestamp{Seconds: 10}},
	}

	// Copied from 'getLastSecretVersion'
	versions = sortVersions(versions)
	assert.Equal(t, int64(42), versions[0].CreateTime.Seconds, "Should put version 42 first")
}
