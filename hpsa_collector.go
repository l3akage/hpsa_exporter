package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const prefix = "hpsa_"

var (
	upDesc            *prometheus.Desc
	commandDesc       *prometheus.Desc
	controllerDesc    *prometheus.Desc
	arrayDesc         *prometheus.Desc
	logicalDriveDesc  *prometheus.Desc
	physicalDriveDesc *prometheus.Desc
)

func init() {
	commandDesc = prometheus.NewDesc(prefix+"up", "Scrape was successful", nil, nil)
	controllerDesc = prometheus.NewDesc(prefix+"controller", "Controller state", []string{"name", "slot", "status", "cache_status", "battery_status"}, nil)
	arrayDesc = prometheus.NewDesc(prefix+"array", "Array state", []string{"ctrl_slot", "name", "type", "unused_space", "status"}, nil)
	logicalDriveDesc = prometheus.NewDesc(prefix+"logical_drive", "Logical drive state", []string{"ctrl_slot", "array", "index", "size", "raid", "status"}, nil)
	physicalDriveDesc = prometheus.NewDesc(prefix+"physical_drive", "Physical drive state", []string{"ctrl_slot", "array", "size", "position", "type", "status"}, nil)
}

func getCmdOutput() ([]byte, error) {
	if *command != "" {
		_, err := exec.LookPath(*command)
		if err != nil {
			log.Fatalf("command %s not found", *command)
		}
	} else {
		commands := []string{"hpacucli", "ssacli"}
		for _, cmd := range commands {
			_, err := exec.LookPath(cmd)
			if err != nil {
				continue
			}
			command = &cmd
			break
		}
	}
	if *command == "" {
		log.Fatal("no command found")
	}

	args := []string{"ctrl", "all", "show", "config", "detail"}
	output, err := exec.Command(*command, args...).Output()
	return output, err
}

func parseLogicalDrive(lines []string, idx int) (*logicalDrive, int) {
	data := strings.Split(lines[idx], ": ")
	d := &logicalDrive{
		Index: data[1],
	}
	for {
		idx++
		line := strings.Trim(lines[idx], " ")
		if len(line) == 0 {
			break
		}
		if strings.HasPrefix(line, "Size: ") {
			d.Size = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Fault Tolerance: ") {
			d.RaidMode = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Status: ") {
			d.Status = strings.Split(line, ": ")[1]
		}
	}
	return d, idx
}

func parsePhysicalDrive(lines []string, idx int) (*physicalDrive, int) {
	data := strings.Split(strings.Trim(lines[idx], " "), " ")
	d := &physicalDrive{
		Position: data[1],
	}
	for {
		idx++
		line := strings.Trim(lines[idx], " ")
		if len(line) == 0 {
			break
		}
		if strings.HasPrefix(line, "Size: ") {
			d.Size = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Status: ") {
			d.Status = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Interface Type: ") {
			d.Type = strings.Split(line, ": ")[1]
		}
	}
	return d, idx
}

func parseArray(lines []string, idx int) (*array, int) {
	data := strings.Split(lines[idx], ": ")
	a := &array{
		Name:           data[1],
		LogicalDrives:  []logicalDrive{},
		PhysicalDrives: []physicalDrive{},
	}
	for {
		idx++
		if idx >= len(lines) {
			break
		}
		line := strings.Trim(lines[idx], " ")
		if strings.Contains(line, " in Slot ") {
			idx--
			break
		} else if strings.HasPrefix(line, "Array: ") {
			idx--
			break
		} else if strings.HasPrefix(line, "Interface Type: ") {
			a.Type = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Unused Space: ") {
			a.UnusedSpace = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Status: ") {
			a.Status = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Logical Drive: ") {
			var d *logicalDrive
			d, idx = parseLogicalDrive(lines, idx)
			a.LogicalDrives = append(a.LogicalDrives, *d)
		} else if strings.HasPrefix(line, "physicaldrive ") {
			var d *physicalDrive
			d, idx = parsePhysicalDrive(lines, idx)
			a.PhysicalDrives = append(a.PhysicalDrives, *d)
		}
	}
	return a, idx
}

func parseController(lines []string, idx int) (*controller, int) {
	data := strings.Split(lines[idx], " in Slot ")
	c := &controller{
		Name:   data[0],
		Slot:   data[1],
		Arrays: []array{},
	}
	for {
		idx++
		if idx >= len(lines) {
			break
		}
		line := strings.Trim(lines[idx], " ")
		if strings.Contains(line, " in Slot ") {
			break
		} else if strings.HasPrefix(line, "Controller Status: ") {
			c.Status = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Cache Status: ") {
			c.CacheStatus = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Battery/Capacitor Status: ") {
			c.BatteryStatus = strings.Split(line, ": ")[1]
		} else if strings.HasPrefix(line, "Array: ") {
			var a *array
			a, idx = parseArray(lines, idx)
			c.Arrays = append(c.Arrays, *a)
		}
	}
	return c, idx
}

func parseOutput(output []byte) ([]*controller, error) {
	var controllers []*controller
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	idx := 0

	for {
		if idx >= len(lines) {
			break
		}
		line := strings.Trim(lines[idx], " ")
		if !strings.Contains(line, " in Slot ") {
			idx++
			continue
		}
		var c *controller
		c, idx = parseController(lines, idx)
		controllers = append(controllers, c)
	}
	return controllers, nil
}

type hpsaCollector struct {
}

func (c hpsaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- commandDesc
	ch <- controllerDesc
	ch <- arrayDesc
	ch <- logicalDriveDesc
	ch <- physicalDriveDesc
}

func (c hpsaCollector) Collect(ch chan<- prometheus.Metric) {
	output, err := getCmdOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error in running hpsa command", err)
		ch <- prometheus.MustNewConstMetric(commandDesc, prometheus.GaugeValue, 0)
	} else {
		controllers, err := parseOutput(output)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing output", err)
			ch <- prometheus.MustNewConstMetric(commandDesc, prometheus.GaugeValue, 0)
		} else {
			ch <- prometheus.MustNewConstMetric(commandDesc, prometheus.GaugeValue, 1)
			for _, controller := range controllers {
				controller.Describe(ch)
			}
		}
	}
}
