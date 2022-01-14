package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Q42/gcp-sema/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
)

type proxyInstance struct {
	port            int
	listGetCounter  int32
	valueGetCounter int32
	emitList        context.CancelFunc
	emitValue       context.CancelFunc
}

// TestProxy has the primary objective of establishing that "proxy" can handle concurrent requests and does not block indefinetely.
func TestProxy(t *testing.T) {
	p := proxyInstance{}
	p.Prepare()
	t.Run("test", func(t *testing.T) {
		p.Run(t)
	})
}

func BenchmarkProxy(t *testing.B) {
	t.Run("test", func(b *testing.B) {
		p := proxyInstance{}
		p.Prepare()
		t.ResetTimer()
		p.Run(b)
	})
}

func (instance *proxyInstance) Prepare() {
	var listCtx, valueCtx context.Context
	listCtx, instance.emitList = context.WithCancel(context.Background())
	valueCtx, instance.emitValue = context.WithCancel(context.Background())
	opts := proxyCommand{Address: ":0", prepareClient: func(projectID string) secretmanager.KVClient {
		return &ctxClient{
			listCtx, valueCtx,
			secretmanager.NewInMemoryClient("test",
				"foo0", "bar",
				"foo1", "bar",
				"foo2", "bar",
				"foo3", "bar"),
			func() { atomic.AddInt32(&instance.listGetCounter, 1) },
			func() { atomic.AddInt32(&instance.valueGetCounter, 1) },
		}
	}}
	go opts.Execute(nil)
	time.Sleep(10 * time.Millisecond)
	instance.port = opts.listener.Addr().(*net.TCPAddr).Port
}

func (instance *proxyInstance) Run(t assert.TestingT) {
	wg := sync.WaitGroup{}
	runInstance := func(i int) {
		resp, err := http.Get(fmt.Sprintf("http://:%d/list?project=test", instance.port))
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, readBody(resp.Body))
		resp, err = http.Get(fmt.Sprintf("http://:%d/get?project=test&shortName=foo%d&fullName=foo%d", instance.port, i, i))
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, readBody(resp.Body))
		wg.Done()
	}
	count := 20
	wg.Add(count)
	for i := 0; i < count; i++ {
		go runInstance(i % 4) // 0-3 five times
	}

	// kick off
	instance.emitList()
	instance.emitValue()
	wg.Wait()
	assert.EqualValues(t, 1, instance.listGetCounter, "should get list only once")
	assert.EqualValues(t, 4, instance.valueGetCounter, "should get each value only once")
}

func readBody(rc io.ReadCloser) string {
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		panic(err)
	}
	return string(data)
}

type ctxClient struct {
	context.Context
	valueCtx context.Context
	delegate secretmanager.KVClient
	OnList   func()
	OnValue  func()
}

var _ secretmanager.KVClient = &ctxClient{}

func (c *ctxClient) ListKeys() (keys []secretmanager.KVValue, err error) {
	c.OnList()
	<-c.Context.Done()
	keys, err = c.delegate.ListKeys()
	for i, key := range keys {
		keys[i] = &ctxValue{KVValue: key, Context: c.valueCtx, OnValue: c.OnValue}
	}
	return
}
func (c *ctxClient) Get(name string) (secretmanager.KVValue, error) {
	kv, err := c.delegate.Get(name)
	return &ctxValue{KVValue: kv, Context: c.valueCtx, OnValue: c.OnValue}, err
}
func (c *ctxClient) New(name string, labels map[string]string) (secretmanager.KVValue, error) {
	return nil, errors.New("unimplemented")
}

type ctxValue struct {
	secretmanager.KVValue
	context.Context
	OnValue func()
}

var _ secretmanager.KVValue = &ctxValue{}

func (c *ctxValue) GetValue() ([]byte, error) {
	c.OnValue()
	<-c.Context.Done()
	return c.KVValue.GetValue()
}
