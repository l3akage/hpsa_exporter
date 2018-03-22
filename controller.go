package main

import "github.com/prometheus/client_golang/prometheus"

type controller struct {
	Name          string
	Slot          string
	Status        string
	CacheStatus   string
	BatteryStatus string
	Arrays        []array
}

func (c controller) Describe(ch chan<- prometheus.Metric) {
	l := []string{c.Name, c.Slot, c.Status, c.CacheStatus, c.BatteryStatus}
	ch <- prometheus.MustNewConstMetric(controllerDesc, prometheus.GaugeValue, 1, l...)
	for _, array := range c.Arrays {
		array.Describe(ch, c)
	}
}
