package main

import "github.com/prometheus/client_golang/prometheus"

type logicalDrive struct {
	Index    string
	Size     string
	RaidMode string
	Status   string
}

func (d logicalDrive) Describe(ch chan<- prometheus.Metric, c controller, a array) {
	l := []string{c.Slot, a.Name, d.Index, d.Size, d.RaidMode, d.Status}
	ch <- prometheus.MustNewConstMetric(logicalDriveDesc, prometheus.GaugeValue, 1, l...)
}
