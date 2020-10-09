package secretmanager

import (
	"context"
	"fmt"
	"sort"
	"strings"

	sema "cloud.google.com/go/secretmanager/apiv1"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/genproto/protobuf/field_mask"
)

// ErrNoVersions means the secret exists but it has no enabled versions
var ErrNoVersions = errors.New("no versions")

// NewClient creates a new wrapped Secret Manager client
func NewClient(project string) (KVClient, error) {
	ctx := context.Background()
	client, err := sema.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return semaWrapper{client, project, ctx}, nil
}

type semaWrapper struct {
	client  *sema.Client
	project string
	ctx     context.Context
}
type semaSecretWrapper struct {
	client *semaWrapper
	path   string // "format: projects/%s/secrets/%s"
	labels map[string]string
}

var _ KVClient = semaWrapper{}
var _ KVValue = semaSecretWrapper{}

func (s semaWrapper) ListKeys() ([]KVValue, error) {
	it := s.client.ListSecrets(s.ctx, &secretmanagerpb.ListSecretsRequest{
		// The parenet resource in the format `projects/*`.
		Parent: fmt.Sprintf("projects/%s", s.project),
	})

	data := make([]KVValue, 0)

	for {
		sec, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data = append(data, KVValue(semaSecretWrapper{client: &s, path: sec.Name, labels: sec.Labels}))
	}

	return data, nil
}

func (s semaWrapper) Get(key string) (KVValue, error) {
	resp, err := s.client.GetSecret(s.ctx, &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", s.project, key),
	})
	if err != nil {
		return nil, err
	}
	return semaSecretWrapper{client: &s, path: resp.Name, labels: resp.Labels}, nil
}

func (s semaWrapper) New(key string, labels map[string]string) (KVValue, error) {
	resp, err := s.client.CreateSecret(s.ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", s.project),
		SecretId: key,
		Secret: &secretmanagerpb.Secret{
			Labels: labels, Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{},
			}},
	})
	_ = resp.Name // TODO use this resp.Name maybe
	if err != nil {
		return nil, err
	}
	return KVValue(semaSecretWrapper{client: &s, path: resp.Name, labels: labels}), nil
}

func (s semaSecretWrapper) GetFullName() string  { return s.path }
func (s semaSecretWrapper) GetShortName() string { return s.path[strings.LastIndex(s.path, "/")+1:] }

func (s semaSecretWrapper) GetLink() string {
	return fmt.Sprintf("https://console.cloud.google.com/security/secret-manager/secret/%s?project=%s", s.GetShortName(), s.client.project)
}

func (s semaSecretWrapper) getLastVersion() (string, error) {
	versions := make([]*secretmanagerpb.SecretVersion, 0)
	it := s.client.client.ListSecretVersions(s.client.ctx, &secretmanagerpb.ListSecretVersionsRequest{Parent: s.path})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		// *secretmanagerpb.SecretVersion
		// TODO do something with the other properties than Name: CreateTime, DestroyTime, State
		if resp.State == secretmanagerpb.SecretVersion_ENABLED {
			versions = append(versions, resp)
		}
	}
	versions = sortVersions(versions)
	if len(versions) == 0 {
		return "", errors.Wrap(ErrNoVersions, fmt.Sprintf(`Secret %q (%s)`, s.GetShortName(), s.GetLink()))
	}
	// The resource name in the format `projects/*/secrets/*/versions/*`.
	return versions[0].Name, nil
}

func (s semaSecretWrapper) GetValue() ([]byte, error) {
	version, err := s.getLastVersion()
	if err != nil {
		return nil, err
	}
	req := &secretmanagerpb.AccessSecretVersionRequest{Name: version}
	resp, err := s.client.client.AccessSecretVersion(s.client.ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Payload.Data, nil
}

func (s semaSecretWrapper) GetLabels() map[string]string { return s.labels }

func (s semaSecretWrapper) SetValue(value []byte) (string, error) {
	// writeSecretVersion updates the value to a new version
	// projectNameVersion is the resource name in the format `projects/*/secrets/*`.
	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent:  s.path,
		Payload: &secretmanagerpb.SecretPayload{Data: value},
	}
	resp, err := s.client.client.AddSecretVersion(s.client.ctx, req)
	if err != nil {
		return "", err
	}
	_ = resp.Name // TODO something with this name (contains new version)
	return resp.Name, err
}

func (s semaSecretWrapper) SetLabels(value map[string]string) error {
	secret, err := s.client.client.GetSecret(s.client.ctx, &secretmanagerpb.GetSecretRequest{
		Name: s.path,
	})
	if err != nil {
		return err
	}
	secret.Labels = value
	secret, err = s.client.client.UpdateSecret(s.client.ctx, &secretmanagerpb.UpdateSecretRequest{
		Secret:     secret,
		UpdateMask: &field_mask.FieldMask{Paths: []string{"labels"}},
	})
	return err
}

func sortVersions(versions []*secretmanagerpb.SecretVersion) []*secretmanagerpb.SecretVersion {
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreateTime.Seconds > versions[j].CreateTime.Seconds
	})
	return versions
}

// SecretShortNames reduces a list of KVValues to a list of short names
func SecretShortNames(list []KVValue) []string {
	var names []string = nil
	for _, s := range list {
		names = append(names, s.GetShortName())
	}
	return names
}
