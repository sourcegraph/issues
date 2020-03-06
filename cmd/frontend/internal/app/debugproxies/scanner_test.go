package debugproxies

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type k8sTestClient struct {
	listResponse *corev1.ServiceList
	getResponses map[string]*corev1.Endpoints
}

func (ktc *k8sTestClient) Watch(ctx context.Context, namespace string, r k8s.Resource, options ...k8s.Option) (*k8s.Watcher, error) {
	// we don't use it for tests yet, once we do we need to mock the returned watcher too
	return nil, errors.New("not implemented")
}

func (ktc *k8sTestClient) List(ctx context.Context, namespace string, resp k8s.ResourceList, options ...k8s.Option) error {
	sxs := resp.(*corev1.ServiceList)

	sxs.Items = ktc.listResponse.Items
	sxs.Metadata = ktc.listResponse.Metadata
	return nil
}

func (ktc *k8sTestClient) Get(ctx context.Context, namespace, name string, resp k8s.Resource, options ...k8s.Option) error {
	ep := ktc.getResponses[name]
	if ep == nil {
		return fmt.Errorf("resource with name %s not set up as fixture", name)
	}

	rep := resp.(*corev1.Endpoints)

	rep.Metadata = ep.Metadata
	rep.Subsets = ep.Subsets
	return nil
}

func (ktc *k8sTestClient) Namespace() string {
	return "foospace"
}

func stringPtr(val string) *string {
	str := val
	return &str
}

func TestClusterScan(t *testing.T) {
	var eps []Endpoint

	consumer := func(seen []Endpoint) {
		eps = nil
		for _, ep := range seen {
			eps = append(eps, ep)
		}
	}

	ktc := &k8sTestClient{
		getResponses: make(map[string]*corev1.Endpoints),
	}

	cs := &clusterScanner{
		client:  ktc,
		consume: consumer,
	}

	ktc.getResponses["gitserver"] = &corev1.Endpoints{
		Subsets: []*corev1.EndpointSubset{{
			Addresses: []*corev1.EndpointAddress{{
				Ip: stringPtr("192.168.10.0"),
			}},
		}},
	}

	ktc.listResponse = &corev1.ServiceList{
		Items: []*corev1.Service{
			{
				Metadata: &metav1.ObjectMeta{
					Namespace: stringPtr("foospace"),
					Name:      stringPtr("gitserver"),
					Annotations: map[string]string{
						"sourcegraph.prometheus/scrape": "true",
						"prometheus.io/port":            "2323",
					},
				},
			},
			{
				Metadata: &metav1.ObjectMeta{
					Namespace: stringPtr("foospace"),
					Name:      stringPtr("no-scrape"),
					Annotations: map[string]string{
						"prometheus.io/port": "2323",
					},
				},
			},
			{
				Metadata: &metav1.ObjectMeta{
					Namespace: stringPtr("foospace"),
					Name:      stringPtr("no-port"),
					Annotations: map[string]string{
						"sourcegraph.prometheus/scrape": "true",
					},
				},
			},
		},
	}

	cs.scanCluster()

	if len(eps) != 1 {
		t.Errorf("expected one found endpoint")
		return
	}

	if eps[0].Service != "gitserver" || eps[0].Host != "192.168.10.0:2323" {
		t.Errorf("expected gitserver-192.168.10.0:2323, got %+v", eps[0])
	}
}
