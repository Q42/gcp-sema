package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/Q42/gcp-sema/pkg/secretmanager/singleflight"
	"github.com/pkg/errors"
)

var proxyDescription = `proxy starts a server which can be used with SEMA_PROXY for the regular commands.`

// proxyCommand defines the options
type proxyCommand struct {
	Address         string `long:"address" default:"127.0.0.1:8080" description:"Listen address. Do not expose this server publicly!"`
	TLSCertFile     string `long:"cert" default:"" description:"If set, starts the server in secure TLS mode."`
	TLSKeyFile      string `long:"key" default:"" description:"If set, starts the server in secure TLS mode."`
	secretListCache map[string][]secretmanager.KVValue
	secretDataCache map[string]func() chan GetValueResult

	secretClients    map[string]secretmanager.KVClient
	secretListCacheM sync.Mutex
	secretDataCacheM sync.Mutex
	secretClientsM   sync.Mutex
	// Testing
	listener      net.Listener
	prepareClient func(projectID string) secretmanager.KVClient
}

func init() {
	_, err := parser.AddCommand("proxy", proxyDescription, proxyDescription, &proxyCommand{})
	panicIfErr(err)
}

func (opts *proxyCommand) Execute(args []string) (err error) {
	opts.secretListCache = make(map[string][]secretmanager.KVValue)
	opts.secretDataCache = make(map[string]func() chan GetValueResult)
	opts.secretClients = make(map[string]secretmanager.KVClient)
	if opts.prepareClient == nil {
		opts.prepareClient = prepareSemaClient
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/list", opts.list)
	mux.HandleFunc("/get", opts.get)
	if opts.TLSKeyFile != "" || opts.TLSCertFile != "" {
		if opts.TLSKeyFile == "" || opts.TLSCertFile == "" {
			return errors.New("If either --cert or --key is set, you must specify both")
		}
		return http.ListenAndServeTLS(opts.Address, opts.TLSCertFile, opts.TLSKeyFile, mux)
	}
	log.Println("Starting insecure gcp-sema proxy server")
	server := http.Server{
		Addr:    opts.Address,
		Handler: mux,
	}
	opts.listener, err = net.Listen("tcp", opts.Address)
	if err != nil {
		return err
	}
	return server.Serve(opts.listener)
}

func (opts *proxyCommand) getClient(projectID string) secretmanager.KVClient {
	opts.secretClientsM.Lock()
	defer opts.secretClientsM.Unlock()

	if c, exists := opts.secretClients[projectID]; exists {
		return c
	}
	client := singleflight.New(opts.prepareClient(projectID))
	opts.secretClients[projectID] = client
	return client
}

func (opts *proxyCommand) getListSafe(projectID string) (keys []secretmanager.KVValue, hit bool, err error) {
	client := opts.getClient(projectID)

	opts.secretListCacheM.Lock()
	defer opts.secretListCacheM.Unlock()

	if listKeys, exists := opts.secretListCache[projectID]; exists {
		hit = true
		keys = listKeys
	} else {
		hit = false
		keys, err = client.ListKeys()
		if err != nil {
			return nil, false, err
		}
		opts.secretListCache[projectID] = keys
	}
	return keys, hit, err
}

func (opts *proxyCommand) getCachedSingleSafe(projectID string, shortName string) (k secretmanager.KVValue, hit bool) {
	opts.secretListCacheM.Lock()
	defer opts.secretListCacheM.Unlock()

	if listCache, exists := opts.secretListCache[projectID]; exists {
		for _, existingSecret := range listCache {
			if existingSecret.GetShortName() == shortName {
				k = existingSecret
			}
		}
	}
	return k, k != nil
}

type GetValueResult struct {
	data []byte
	err  error
}

func (opts *proxyCommand) getValueSafe(projectID string, k secretmanager.KVValue) (data []byte, hit bool, err error) {
	opts.secretDataCacheM.Lock()
	d, exists := opts.secretDataCache[k.GetFullName()]
	if exists {
		opts.secretDataCacheM.Unlock()
		hit = true
	} else {
		hit = false
		var loaded bool
		var cache []byte
		var err error
		start := make(chan struct{}, 1)
		done := make(chan GetValueResult)
		go func() {
			for {
				<-start
				if loaded {
					done <- GetValueResult{cache, err}
					continue
				}
				cache, err = k.GetValue()
				loaded = err == nil
				done <- GetValueResult{cache, err}
			}
		}()
		d = func() chan GetValueResult {
			start <- struct{}{}
			return done
		}
		opts.secretDataCache[k.GetFullName()] = d
		opts.secretDataCacheM.Unlock()
	}

	result := <-d()
	return result.data, hit, result.err
}

func (opts *proxyCommand) list(rw http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	log.Printf("Retrieving from %s", projectID)
	var err error

	// Get keys in project

	keys, hit, err := opts.getListSafe(projectID)
	if err != nil {
		log.Println(err)
		http.Error(rw, err.Error(), 500)
		return
	}
	if hit {
		rw.Header().Add("Cache-Hit", "true")
	} else {
		rw.Header().Add("Cache-Hit", "false")
	}

	log.Printf("Retrieved %d keys from %s", len(keys), projectID)
	list := proxyListing{}
	for _, k := range keys {
		list.Secrets = append(list.Secrets, proxySecret{
			FullName:  k.GetFullName(),
			ShortName: k.GetShortName(),
			Labels:    k.GetLabels(),
		})
	}

	data, err := json.Marshal(&list)
	if err != nil {
		log.Println(err)
		http.Error(rw, err.Error(), 500)
		return
	}

	rw.WriteHeader(200)
	rw.Write(data)
}

func (opts *proxyCommand) get(rw http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	shortName := r.URL.Query().Get("shortName")
	fullName := r.URL.Query().Get("fullName")

	// Get secret
	var err error
	k, _ := opts.getCachedSingleSafe(projectID, shortName)
	if k == nil {
		client := opts.getClient(projectID)
		k, err = client.Get(shortName)
		if err != nil {
			log.Println(err)
			http.Error(rw, err.Error(), 500)
			return
		}
	}

	// Get the secret data payload
	data, hit, err := opts.getValueSafe(projectID, k)
	if err != nil {
		log.Println(err)
		http.Error(rw, err.Error(), 500)
		return
	}
	if hit {
		rw.Header().Add("Cache-Hit", "true")
	} else {
		rw.Header().Add("Cache-Hit", "false")
	}

	log.Printf("Sending %s", fullName)
	jsonData, err := json.Marshal(proxySecretDetail{
		ProxySecret: proxySecret{
			ShortName: k.GetShortName(),
			FullName:  k.GetFullName(),
			Labels:    k.GetLabels(),
		},
		Data: base64.RawStdEncoding.EncodeToString(data),
	})
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	rw.WriteHeader(200)
	rw.Write(jsonData)
}

// NewProxyClient can be used instead of a regular Secret Manager client. It uses the proxy server.
func NewProxyClient(proxyAddr string, project string) secretmanager.KVClient {
	return proxyClient{proxyAddr: proxyAddr, project: project, secret: nil}
}

type proxyClient struct {
	proxyAddr string
	project   string
	secret    *proxySecret
}

type proxyListing struct {
	Secrets []proxySecret
}

type proxySecret struct {
	ShortName string
	FullName  string
	Labels    map[string]string
}

type proxySecretDetail struct {
	ProxySecret proxySecret
	Data        string
}

func (c proxyClient) ListKeys() (result []secretmanager.KVValue, err error) {
	list := proxyListing{}
	err = jsonReq(fmt.Sprintf("%s/list?project=%s", c.proxyAddr, url.QueryEscape(c.project)), &list)
	if err != nil {
		return nil, errors.Wrap(err, "proxy/list failed")
	}
	for _, s := range list.Secrets {
		var copied = s // new assignment
		result = append(result, proxyClient{c.proxyAddr, c.project, &copied})
	}
	return result, nil
}

func (c proxyClient) Get(name string) (secretmanager.KVValue, error) {
	keys, err := c.ListKeys()
	if err != nil {
		return nil, errors.Wrap(err, "proxy/get failed")
	}
	for _, k := range keys {
		if k.GetShortName() == name || k.GetFullName() == name {
			return k, nil
		}
	}
	return nil, fmt.Errorf("secret not found %q", name)
}

func (c proxyClient) New(name string, labels map[string]string) (secretmanager.KVValue, error) {
	return nil, errors.New("Proxy is a read-only implementation. Do not use --proxy to make edits")
}

func (c proxyClient) GetFullName() string                      { return c.secret.FullName }
func (c proxyClient) GetShortName() string                     { return c.secret.ShortName }
func (c proxyClient) GetLabels() map[string]string             { return c.secret.Labels }
func (c proxyClient) SetLabels(labels map[string]string) error { return errors.New("Readonly") }
func (c proxyClient) SetValue([]byte) (string, error)          { return "", errors.New("Readonly") }

func (c proxyClient) GetValue() ([]byte, error) {
	detail := proxySecretDetail{}
	err := jsonReq(fmt.Sprintf("%s/get?project=%s&shortName=%s&fullName=%s", c.proxyAddr,
		url.QueryEscape(c.project),
		url.QueryEscape(c.secret.ShortName),
		url.QueryEscape(c.secret.FullName),
	), &detail)
	if err != nil {
		return nil, errors.Wrap(err, "proxy/get failed")
	}
	data, err := base64.RawStdEncoding.DecodeString(detail.Data)
	if err != nil {
		return nil, errors.Wrap(err, "proxy base64 failed")
	}
	return data, nil
}

func jsonReq(url string, dst interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request status not ok: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "body parsing failed")
	}

	err = json.Unmarshal(bytes, dst)
	if err != nil {
		return errors.Wrap(err, "body parsing failed")
	}
	return nil
}
