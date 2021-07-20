package monitoring

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/cockroachdb/errors"
)

const (
	canonicalAlertSolutionsURL = "https://docs.sourcegraph.com/admin/observability/alert_solutions"
	canonicalDashboardsDocsURL = "https://docs.sourcegraph.com/admin/observability/dashboards"

	alertSolutionsFile = "alert_solutions.md"
	dashboardsDocsFile = "dashboards.md"
)

const alertSolutionsHeader = `# Alert solutions

<!-- DO NOT EDIT: generated via: go generate ./monitoring -->

This document contains possible solutions for when you find alerts are firing in Sourcegraph's monitoring.
If your alert isn't mentioned here, or if the solution doesn't help, [contact us](mailto:support@sourcegraph.com) for assistance.

To learn more about Sourcegraph's alerting and how to set up alerts, see [our alerting guide](https://docs.sourcegraph.com/admin/observability/alerting).

`

const dashboardsHeader = `# Dashboards reference

<!-- DO NOT EDIT: generated via: go generate ./monitoring -->

This document contains a complete reference on Sourcegraph's available dashboards, as well as details on how to interpret the panels and metrics.

To learn more about Sourcegraph's metrics and how to view these dashboards, see [our metrics guide](https://docs.sourcegraph.com/admin/observability/metrics).

`

// fprintSubtitle prints subtitle-class text
func fprintSubtitle(w io.Writer, text string) {
	fmt.Fprintf(w, "<p class=\"subtitle\">%s</p>\n\n", text)
}

// Write a standardized Observable header that one can reliably generate an anchor link for.
//
// See `observableAnchor`.
func fprintObservableHeader(w io.Writer, c *Container, o *Observable, headerLevel int) {
	fmt.Fprint(w, strings.Repeat("#", headerLevel))
	fmt.Fprintf(w, " %s: %s\n\n", c.Name, o.Name)
}

// fprintOwnedBy prints information about who owns a particular monitoring definition.
func fprintOwnedBy(w io.Writer, owner ObservableOwner) {
	fmt.Fprintf(w, "<sub>*Managed by the %s.*</sub>\n", owner.toMarkdown())
}

// Create an anchor link that matches `fprintObservableHeader`
//
// Must match Prometheus template in `docker-images/prometheus/cmd/prom-wrapper/receivers.go`
func observableDocAnchor(c *Container, o Observable) string {
	observableAnchor := strings.ReplaceAll(o.Name, "_", "-")
	return fmt.Sprintf("%s-%s", c.Name, observableAnchor)
}

type documentation struct {
	alertSolutions bytes.Buffer
	dashboards     bytes.Buffer
}

func renderDocumentation(containers []*Container) (*documentation, error) {
	var docs documentation

	fmt.Fprint(&docs.alertSolutions, alertSolutionsHeader)
	fmt.Fprint(&docs.dashboards, dashboardsHeader)

	for _, c := range containers {
		fmt.Fprintf(&docs.dashboards, "## %s\n\n", c.Title)
		fprintSubtitle(&docs.dashboards, c.Description)

		for _, g := range c.Groups {
			// the "General" group is top-level
			if g.Title != "General" {
				fmt.Fprintf(&docs.dashboards, "### %s: %s\n\n", c.Title, g.Title)
			}

			for _, r := range g.Rows {
				for _, o := range r {
					if err := docs.renderAlertSolutionEntry(c, o); err != nil {
						return nil, errors.Errorf("error rendering alert solution entry %q %q: %w",
							c.Name, o.Name, err)
					}
					if err := docs.renderDashboardPanelEntry(c, o); err != nil {
						return nil, errors.Errorf("error rendering dashboard panel entry  %q %q: %w",
							c.Name, o.Name, err)
					}
				}
			}
		}
	}

	return &docs, nil
}

func (d *documentation) renderAlertSolutionEntry(c *Container, o Observable) error {
	if o.Warning == nil && o.Critical == nil {
		return nil
	}

	fprintObservableHeader(&d.alertSolutions, c, &o, 2)
	fprintSubtitle(&d.alertSolutions, o.Description)

	var prometheusAlertNames []string // collect names for silencing configuration
	// Render descriptions of various levels of this alert
	fmt.Fprintf(&d.alertSolutions, "**Descriptions**\n\n")
	for _, alert := range []struct {
		level     string
		threshold *ObservableAlertDefinition
	}{
		{level: "warning", threshold: o.Warning},
		{level: "critical", threshold: o.Critical},
	} {
		if alert.threshold.isEmpty() {
			continue
		}
		desc, err := c.alertDescription(o, alert.threshold)
		if err != nil {
			return err
		}
		fmt.Fprintf(&d.alertSolutions, "- <span class=\"badge badge-%s\">%s</span> %s\n", alert.level, alert.level, desc)
		prometheusAlertNames = append(prometheusAlertNames,
			fmt.Sprintf("  \"%s\"", prometheusAlertName(alert.level, c.Name, o.Name)))
	}
	fmt.Fprint(&d.alertSolutions, "\n")

	// Render solutions for dealing with this alert
	fmt.Fprintf(&d.alertSolutions, "**Possible solutions**\n\n")
	if o.PossibleSolutions != "none" {
		possibleSolutions, _ := toMarkdown(o.PossibleSolutions, true)
		fmt.Fprintf(&d.alertSolutions, "%s\n", possibleSolutions)
	}
	// add silencing configuration as another solution
	fmt.Fprintf(&d.alertSolutions, "- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:\n\n")
	fmt.Fprintf(&d.alertSolutions, "```json\n%s\n```\n\n", fmt.Sprintf(`"observability.silenceAlerts": [
%s
]`, strings.Join(prometheusAlertNames, ",\n")))
	// add link to panel information IF there are additional details available
	if o.Interpretation != "" && o.Interpretation != "none" {
		fmt.Fprintf(&d.alertSolutions, "> NOTE: More help interpreting this metric is available in the [dashboards reference](./%s#%s).\n\n",
			dashboardsDocsFile, observableDocAnchor(c, o))
	}
	if o.Owner != "" {
		// add owner
		fprintOwnedBy(&d.alertSolutions, o.Owner)
	}
	// render break for readability
	fmt.Fprint(&d.alertSolutions, "\n<br />\n\n")
	return nil
}

func (d *documentation) renderDashboardPanelEntry(c *Container, o Observable) error {
	fprintObservableHeader(&d.dashboards, c, &o, 4)
	fmt.Fprintf(&d.dashboards, "This panel indicates %s.\n\n", o.Description)
	// render interpretation reference if available
	if o.Interpretation != "" && o.Interpretation != "none" {
		interpretation, _ := toMarkdown(o.Interpretation, false)
		fmt.Fprintf(&d.dashboards, "%s\n\n", interpretation)
	}
	// add link to alert solutions IF there is an alert attached
	if !o.NoAlert {
		fmt.Fprintf(&d.dashboards, "> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./%s#%s).\n\n",
			alertSolutionsFile, observableDocAnchor(c, o))
	}
	if o.Owner != "" {
		// add owner
		fprintOwnedBy(&d.dashboards, o.Owner)
	}
	// render break for readability
	fmt.Fprint(&d.dashboards, "\n<br />\n\n")
	return nil
}
