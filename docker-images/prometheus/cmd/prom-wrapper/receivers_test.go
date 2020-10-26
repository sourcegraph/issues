package main

import (
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/schema"
)

func TestNewRoutesAndReceivers(t *testing.T) {
	type args struct {
		newAlerts []*schema.ObservabilityAlerts
	}
	tests := []struct {
		name          string
		args          args
		wantProblems  []string
		wantReceivers int
		wantRoutes    int
	}{
		{
			name: "invalid notifier",
			args: args{
				newAlerts: []*schema.ObservabilityAlerts{{
					Level:    "warning",
					Notifier: schema.Notifier{},
				}},
			},
			wantProblems:  []string{"no configuration found"},
			wantReceivers: 2,
			wantRoutes:    2,
		},
		{
			name: "one warning one critical",
			args: args{
				newAlerts: []*schema.ObservabilityAlerts{{
					Level: "warning",
					Notifier: schema.Notifier{
						Slack: &schema.NotifierSlack{
							Type: "slack",
							Url:  "https://sourcegraph.com",
						},
					},
				}, {
					Level: "critical",
					Notifier: schema.Notifier{
						Slack: &schema.NotifierSlack{
							Type: "slack",
							Url:  "https://sourcegraph.com",
						},
					},
				}},
			},
			wantProblems:  nil,
			wantReceivers: 2,
			wantRoutes:    2,
		}, {
			name: "one custom route",
			args: args{
				newAlerts: []*schema.ObservabilityAlerts{{
					Level: "warning",
					Notifier: schema.Notifier{
						Slack: &schema.NotifierSlack{
							Type: "slack",
							Url:  "https://sourcegraph.com",
						},
					},
					Owners: []string{"distribution"},
				}},
			},
			wantProblems:  nil,
			wantReceivers: 3,
			wantRoutes:    3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := []string{}
			receivers, routes := newRoutesAndReceivers(tt.args.newAlerts, "https://sourcegraph.com", func(err error) {
				problems = append(problems, err.Error())
			})
			if len(tt.wantProblems) != len(problems) {
				t.Errorf("expected problems %+v, got %+v", tt.wantProblems, problems)
				return
			}
			for i, p := range problems {
				if !strings.Contains(p, tt.wantProblems[i]) {
					t.Errorf("expected problem %v to contain %q, got %q", i, tt.wantProblems[i], p)
					return
				}
			}
			if len(receivers) != tt.wantReceivers {
				t.Errorf("expected %d receivers, got %d", tt.wantReceivers, len(receivers))
				return
			}
			if len(routes) != tt.wantRoutes {
				t.Errorf("expected %d routes, got %d", tt.wantRoutes, len(routes))
				return
			}

			// check each route has valid receiver
			receiverNames := map[string]struct{}{}
			for _, rc := range receivers {
				receiverNames[rc.Name] = struct{}{}
			}
			for i, rt := range routes {
				if _, receiverExists := receiverNames[rt.Receiver]; !receiverExists {
					t.Errorf("route %d uses receiver %q, but receiver does not exist", i, rt.Receiver)
				}
			}
		})
	}
}
