package main

import "github.com/prometheus/client_golang/prometheus"

type array struct {
	Name           string
	Type           string
	UnusedSpace    string
	Status         string
	LogicalDrives  []logicalDrive
	PhysicalDrives []physicalDrive
}

func (a array) Describe(ch chan<- prometheus.Metric, c controller) {
	l := []string{c.Slot, a.Name, a.Type, a.UnusedSpace, a.Status}
	ch <- prometheus.MustNewConstMetric(arrayDesc, prometheus.GaugeValue, 1, l...)
	for _, drive := range a.LogicalDrives {
		drive.Describe(ch, c, a)
	}
	for _, drive := range a.PhysicalDrives {
		drive.Describe(ch, c, a)
	}
}
