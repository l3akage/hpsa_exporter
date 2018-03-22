package main

import "github.com/prometheus/client_golang/prometheus"

type physicalDrive struct {
	Size     string
	Status   string
	Type     string
	Position string
}

func (d physicalDrive) Describe(ch chan<- prometheus.Metric, c controller, a array) {
	l := []string{c.Slot, a.Name, d.Size, d.Position, d.Status}
	ch <- prometheus.MustNewConstMetric(physicalDriveDesc, prometheus.GaugeValue, 1, l...)
}
