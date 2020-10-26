// Package endpoint provides a consistent hash map for URLs to kubernetes
// endpoints.
package endpoint

import (
	"context"
	"fmt"
	"hash/crc32"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/pkg/errors"
)

// Map is a consistent hash map to URLs. It uses the kubernetes API to watch
// the endpoints for a service and update the map when they change. It can
// also fallback to static URLs if not configured for kubernetes.
type Map struct {
	mu      sync.Mutex
	init    func() (*hashMap, error)
	err     error
	urls    *hashMap
	urlspec string
}

// New creates a new Map for the URL specifier.
//
// If the scheme is prefixed with "k8s+", one URL is expected and the format is
// expected to match e.g. k8s+http://service.namespace:port/path. namespace,
// port and path are optional. URLs of this form will consistently hash among
// the endpoints for the Kubernetes service. The values returned by Get will
// look like http://endpoint:port/path.
//
// If the scheme is not prefixed with "k8s+", a space separated list of URLs is
// expected. The map will consistently hash against these URLs in this case.
// This is useful for specifying non-Kubernetes endpoints.
//
// Examples URL specifiers:
//
// 	"k8s+http://searcher"
// 	"http://searcher-1 http://searcher-2 http://searcher-3"
//
func New(urlspec string) *Map {
	if !strings.HasPrefix(urlspec, "k8s+") {
		return &Map{
			urlspec: urlspec,
			urls:    newConsistentHashMap(strings.Fields(urlspec)),
		}
	}

	m := &Map{urlspec: urlspec}

	// Kick off setting the initial urls or err on first access. We don't rely
	// just on inform since it may not communicate updates.
	m.init = func() (*hashMap, error) {
		u, err := parseURL(urlspec)
		if err != nil {
			return nil, err
		}

		client, err := loadClient()
		if err != nil {
			return nil, err
		}

		var endpoints corev1.Endpoints
		err = client.Get(context.Background(), client.Namespace, u.Service, &endpoints)
		if err != nil {
			return nil, err
		}

		// Kick off watcher in the background
		go func() {
			for {
				err := inform(client, m, u)
				log15.Debug("failed to watch kubernetes endpoint", "name", u.Service, "error", err)
				time.Sleep(time.Second)
			}
		}()

		return endpointsToMap(u, &endpoints)
	}

	return m
}

// Static returns an Endpoint map which consistently hashes over endpoints.
//
// There are no requirements on endpoints, it can be any arbitrary
// string. Unlike static endpoints created via New.
//
// Static Maps are guaranteed to never return an error.
func Static(endpoints ...string) *Map {
	return &Map{
		urlspec: fmt.Sprintf("%v", endpoints),
		urls:    newConsistentHashMap(endpoints),
	}
}

// Empty returns an Endpoint map which always fails with err.
func Empty(err error) *Map {
	return &Map{
		urlspec: "error: " + err.Error(),
		err:     err,
	}
}

func (m *Map) String() string {
	return fmt.Sprintf("endpoint.Map(%s)", m.urlspec)
}

// Get the closest URL in the hash to the provided key that is not in
// exclude. If no URL is found, "" is returned.
//
// Note: For k8s URLs we return URLs based on the registered endpoints. The
// endpoint may not actually be available yet / at the moment. So users of the
// URL should implement a retry strategy.
func (m *Map) Get(key string, exclude map[string]bool) (string, error) {
	urls, err := m.getUrls()
	if err != nil {
		return "", err
	}

	return urls.get(key, exclude), nil
}

// GetMany is the same as calling Get on each item of keys. It will only
// acquire the underlying endpoint map once, so is preferred to calling Get
// for each key which will acquire the endpoint map for each call. The benefit
// is it is faster (O(1) mutex acquires vs O(n)) and consistent (endpoint map
// is immutable vs may change between Get calls).
func (m *Map) GetMany(keys ...string) ([]string, error) {
	urls, err := m.getUrls()
	if err != nil {
		return nil, err
	}

	vals := make([]string, len(keys))
	for i := range keys {
		vals[i] = urls.get(keys[i], nil)
	}
	return vals, nil
}

// Endpoints returns a set of all addresses. Do not modify the returned value.
func (m *Map) Endpoints() (map[string]struct{}, error) {
	urls, err := m.getUrls()
	if err != nil {
		return nil, err
	}

	return urls.values, nil
}

func (m *Map) getUrls() (*hashMap, error) {
	m.mu.Lock()
	if m.init != nil {
		m.urls, m.err = m.init()
		m.init = nil // prevent running again
	}
	urls, err := m.urls, m.err
	m.mu.Unlock()
	return urls, err
}

func inform(client *k8s.Client, m *Map, u *k8sURL) error {
	watcher, err := client.Watch(context.Background(), client.Namespace, new(corev1.Endpoints), k8s.QueryParam("fieldSelector", "metadata.name="+u.Service))
	if err != nil {
		return errors.Wrap(err, "client.Watch")
	}
	defer watcher.Close()

	for {
		var endpoints corev1.Endpoints
		eventType, err := watcher.Next(&endpoints)
		if err != nil {
			return errors.Wrap(err, "watcher.Next")
		}

		if eventType != k8s.EventAdded && eventType != k8s.EventModified {
			// Either we are error or the endpoint has been removed.
			log15.Warn(`eventType is not "added" or "modified"`, "eventType", eventType, "subsets", endpoints.Subsets)
			endpoints.Subsets = nil
		}
		urls, err := endpointsToMap(u, &endpoints)
		m.mu.Lock()
		m.urls, m.err = urls, err
		m.mu.Unlock()
	}
}

func endpointsToMap(u *k8sURL, eps *corev1.Endpoints) (*hashMap, error) {
	var urls []string
	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.Hostname != nil && *addr.Hostname != "" {
				urls = append(urls, u.endpointURL(*addr.Hostname+"."+u.Service))
			} else if addr.Ip != nil {
				urls = append(urls, u.endpointURL(*addr.Ip))
			}
		}
	}
	sort.Strings(urls)
	log15.Debug("kubernetes endpoints", "service", u.Service, "urls", urls)
	metricEndpointSize.WithLabelValues(u.Service).Set(float64(len(urls)))
	if len(urls) == 0 {
		return nil, errors.Errorf("no %s endpoints could be found (this may indicate more %s replicas are needed, contact support@sourcegraph.com for assistance)", u.Service, u.Service)
	}
	return newConsistentHashMap(urls), nil
}

type k8sURL struct {
	url.URL

	Service   string
	Namespace string
}

func (u *k8sURL) endpointURL(endpoint string) string {
	uCopy := u.URL
	if port := u.Port(); port != "" {
		uCopy.Host = endpoint + ":" + port
	} else {
		uCopy.Host = endpoint
	}
	if uCopy.Scheme == "rpc" {
		return uCopy.Host
	}
	return uCopy.String()
}

func parseURL(rawurl string) (*k8sURL, error) {
	u, err := url.Parse(strings.TrimPrefix(rawurl, "k8s+"))
	if err != nil {
		return nil, err
	}
	parts := strings.Split(u.Hostname(), ".")
	var svc, ns string
	switch len(parts) {
	case 1:
		svc = parts[0]
	case 2:
		svc, ns = parts[0], parts[1]
	default:
		return nil, fmt.Errorf("invalid k8s url. expected k8s+http://service.namespace:port/path, got %s", rawurl)
	}
	return &k8sURL{
		URL:       *u,
		Service:   svc,
		Namespace: ns,
	}, nil
}

func newConsistentHashMap(keys []string) *hashMap {
	// 50 replicas and crc32.ChecksumIEEE are the defaults used by
	// groupcache.
	m := hashMapNew(50, crc32.ChecksumIEEE)
	m.add(keys...)
	return m
}

func loadClient() (*k8s.Client, error) {
	// Uncomment below to test against a real cluster. This is only important
	// when you are changing how we interact with the k8s API and you want to
	// test against the real thing. Remember to run a proxy to the API first
	// (takes care of auth and endpoint details)
	//
	//   kubectl proxy --port 10810
	//
	// NewInClusterClient only works when running inside of a pod in a k8s
	// cluster.
	/*
		return &k8s.Client{
			Endpoint:  "http://127.0.0.1:10810",
			Namespace: "prod",
			Client:    http.DefaultClient,
			//Client: &http.Client{Transport: &loghttp.Transport{}},
		}, nil
	*/
	return k8s.NewInClusterClient()
}

var metricEndpointSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "src_endpoint_k8s_size",
	Help: "The number of urls in a watched kubernetes endpoint",
}, []string{"service"})
