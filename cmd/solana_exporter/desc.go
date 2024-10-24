package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

type GaugeDesc struct {
	Desc           *prometheus.Desc
	Name           string
	Help           string
	VariableLabels []string
}

func NewGaugeDesc(name string, description string, variableLabels ...string) *GaugeDesc {
	return &GaugeDesc{
		Desc:           prometheus.NewDesc(name, description, variableLabels, nil),
		Name:           name,
		Help:           description,
		VariableLabels: variableLabels,
	}
}

func (c *GaugeDesc) MustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	if len(labels) != len(c.VariableLabels) {
		klog.Fatalf("Provided labels (%v) do not match %s labels (%v)", labels, c.Name, c.VariableLabels)
	}
	return prometheus.MustNewConstMetric(c.Desc, prometheus.GaugeValue, value, labels...)
}

func (c *GaugeDesc) NewInvalidMetric(err error) prometheus.Metric {
	return prometheus.NewInvalidMetric(c.Desc, err)
}
