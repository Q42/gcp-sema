package main

import (
	// Secret Manager API from Google

	"fmt"
	"sort"

	"google.golang.org/api/iterator"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func getAllSecretsInProject(project string) []string {
	it := client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		// The parenet resource in the format `projects/*`.
		Parent: fmt.Sprintf("projects/%s", project),
	})

	data := make([]string, 0)

	for {
		sec, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panicIfErr(err)
		}

		data = append(data, sec.Name)
	}

	return data
}

// Name should have the format projects/*/secrets/*
func getLastSecretVersion(name string) string {
	versions := make([]*secretmanagerpb.SecretVersion, 0)
	it := client.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{Parent: name})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		panicIfErr(err)
		// TODO: Use resp.
		var v *secretmanagerpb.SecretVersion = resp
		// Other properties than Name: CreateTime, DestroyTime, State
		if v.State == secretmanagerpb.SecretVersion_ENABLED {
			versions = append(versions, v)
		}
	}
	versions = sortVersions(versions)
	return versions[0].Name
}

func sortVersions(versions []*secretmanagerpb.SecretVersion) []*secretmanagerpb.SecretVersion {
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreateTime.Seconds > versions[j].CreateTime.Seconds
	})
	return versions
}

// Name should have the format projects/*/secrets/*/version/*
func getSecretValue(projectNameVersion string) *secretmanagerpb.SecretPayload {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		// The resource name in the format `projects/*/secrets/*/versions/*`.
		Name: projectNameVersion,
	}
	resp, err := client.AccessSecretVersion(ctx, req)
	panicIfErr(err)
	return resp.Payload
}
