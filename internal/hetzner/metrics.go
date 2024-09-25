package hetzner

import "github.com/bsm/openmetrics"

type metrics struct {
	reg *openmetrics.Registry

	added    openmetrics.CounterFamily
	deleted  openmetrics.CounterFamily
	modified openmetrics.CounterFamily

	addedFailed    openmetrics.CounterFamily
	deletedFailed  openmetrics.CounterFamily
	modifiedFailed openmetrics.CounterFamily
}

func NewMetrics(reg *openmetrics.Registry) *metrics {
	m := metrics{
		reg: reg,
	}

	m.added = reg.Counter(openmetrics.Desc{
		Name:   "added_records",
		Labels: []string{"domain"},
	})

	m.deleted = reg.Counter(openmetrics.Desc{
		Name:   "deleted_records",
		Labels: []string{"domain"},
	})

	m.modified = reg.Counter(openmetrics.Desc{
		Name:   "modified_records",
		Labels: []string{"domain"},
	})

	m.addedFailed = reg.Counter(openmetrics.Desc{
		Name:   "added_records_failed",
		Labels: []string{"domain"},
	})

	m.deletedFailed = reg.Counter(openmetrics.Desc{
		Name:   "deleted_records_failed",
		Labels: []string{"domain"},
	})

	m.modifiedFailed = reg.Counter(openmetrics.Desc{
		Name:   "modified_records_failed",
		Labels: []string{"domain"},
	})

	return &m
}
