package main

import (
	// Secret Manager API from Google

	"fmt"
	"sort"

	"google.golang.org/api/iterator"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// GcloudProject is used by these Secret Manager utilities. Set it before using them!
var GcloudProject = ""

func getAllSecretsInProject() []string {
	it := client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		// The parenet resource in the format `projects/*`.
		Parent: fmt.Sprintf("projects/%s", GcloudProject),
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

func createSecret(project, name string, labels map[string]string) (string, error) {
	resp, err := client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", project),
		SecretId: name,
		Secret: &secretmanagerpb.Secret{
			Labels: labels, Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{},
			}},
	})
	if err != nil {
		return "", err
	}
	return resp.Name, err
}

// writeSecretVersion updates the value to a new version
// projectNameVersion is the resource name in the format `projects/*/secrets/*`.
func writeSecretVersion(project, secretName, value string) (string, error) {
	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent:  fmt.Sprintf("projects/%s/secrets/%s", project, secretName),
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	}
	resp, err := client.AddSecretVersion(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Name, err
}
