package monitoring

import (
	"fmt"

	"github.com/grafana-tools/sdk"
)

// ObservablePanel declares options for visualizing an Observable, as well as some default
// customization options. A default panel can be instantiated with the `Panel()` constructor,
// and further customized using `ObservablePanel.With(ObservablePanelOption)`.
type ObservablePanel struct {
	options []ObservablePanelOption

	// panelType defines the type of panel
	panelType PanelType

	// unitType is used by other parts of the generator
	unitType UnitType
}

// PanelType denotes the type of the panel's visualization.
//
// Note that this affects `*sdk.Panel` usage in `ObservablePanelOption`s - the value that
// must be modified for changes to apply has to be `p.GraphPanel` or `p.HeatmapPanel`, for example.
// When adding new `PanelType`s, ensure all `ObservablePanelOption`s in this package are
// compatible with each supported type.
type PanelType string

const (
	PanelTypeGraph   PanelType = "graph"
	PanelTypeHeatmap PanelType = "heatmap"
)

func (pt PanelType) validate() bool {
	switch pt {
	case PanelTypeGraph, PanelTypeHeatmap:
		return true
	default:
		return false
	}
}

// Panel provides a builder for customizing an Observable visualization, starting
// with recommended defaults.
func Panel() ObservablePanel {
	return ObservablePanel{
		panelType: PanelTypeGraph,
		options: []ObservablePanelOption{
			PanelOptions.basicPanel(), // required basic values
			PanelOptions.OpinionatedGraphPanelDefaults(),
			PanelOptions.AlertThresholds(),
		},
	}
}

// PanelHeatmap provides a builder for customizing an Observable visualization starting
// with an extremely minimal heatmap panel.
func PanelHeatmap() ObservablePanel {
	return ObservablePanel{
		panelType: PanelTypeHeatmap,
		options: []ObservablePanelOption{
			PanelOptions.basicPanel(), // required basic values
		},
	}
}

// Min sets the minimum value of the Y axis on the panel. The default is zero.
func (p ObservablePanel) Min(min float64) ObservablePanel {
	p.options = append(p.options, func(o Observable, p *sdk.Panel) {
		p.GraphPanel.Yaxes[0].Min = sdk.NewFloatString(min)
	})
	return p
}

// MinAuto sets the minimum value of the Y axis on the panel to auto, instead of
// the default zero.
//
// This is generally only useful if trying to show negative numbers.
func (p ObservablePanel) MinAuto() ObservablePanel {
	p.options = append(p.options, func(o Observable, p *sdk.Panel) {
		p.GraphPanel.Yaxes[0].Min = nil
	})
	return p
}

// Max sets the maximum value of the Y axis on the panel. The default is auto.
func (p ObservablePanel) Max(max float64) ObservablePanel {
	p.options = append(p.options, func(o Observable, p *sdk.Panel) {
		p.GraphPanel.Yaxes[0].Max = sdk.NewFloatString(max)
	})
	return p
}

// LegendFormat sets the panel's legend format, which may use Go template strings to select
// labels from the Prometheus query.
func (p ObservablePanel) LegendFormat(format string) ObservablePanel {
	p.options = append(p.options, func(o Observable, p *sdk.Panel) {
		p.GraphPanel.Targets[0].LegendFormat = format
	})
	return p
}

// Unit sets the panel's Y axis unit type.
func (p ObservablePanel) Unit(t UnitType) ObservablePanel {
	p.unitType = t
	p.options = append(p.options, func(o Observable, p *sdk.Panel) {
		p.GraphPanel.Yaxes[0].Format = string(t)
	})
	return p
}

// Interval declares the panel's interval in milliseconds.
func (p ObservablePanel) Interval(ms int) ObservablePanel {
	p.options = append(p.options, func(o Observable, p *sdk.Panel) {
		p.GraphPanel.Targets[0].Interval = fmt.Sprintf("%dms", ms)
	})
	return p
}

// With adds the provided options to be applied when building this panel.
//
// Before using this, check if the customization you want is already included in the
// default `Panel()` or available as a function on `ObservablePanel`, such as
// `ObservablePanel.Unit(UnitType)` for setting the units on a panel.
//
// Shared customizations are exported by `PanelOptions`, or you can write your option -
// see `ObservablePanelOption` documentation for more details.
func (p ObservablePanel) With(ops ...ObservablePanelOption) ObservablePanel {
	p.options = append(p.options, ops...)
	return p
}

// build applies the configured options on this panel for the given `Observable`.
func (p ObservablePanel) build(o Observable, panel *sdk.Panel) {
	for _, opt := range p.options {
		opt(o, panel)
	}
}
