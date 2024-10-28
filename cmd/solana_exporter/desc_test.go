package main

import (
	"fmt"
	"sort"
)

type LV struct {
	labels []string
	value  float64
}

func NewLV(value float64, labels ...string) LV {
	return LV{labels, value}
}

func (c *GaugeDesc) expectedCollection(labeledValues ...LV) string {
	helpLine := fmt.Sprintf("# HELP %s %s", c.Name, c.Help)
	typeLine := fmt.Sprintf("# TYPE %s gauge", c.Name)
	result := fmt.Sprintf("%s\n%s", helpLine, typeLine)

	// we need to sort our variable labels:
	sortedVariableLabels := make([]string, len(c.VariableLabels))
	copy(sortedVariableLabels, c.VariableLabels)
	sort.Strings(sortedVariableLabels)

	for _, lv := range labeledValues {
		description := ""
		if len(lv.labels) > 0 {
			for i, label := range lv.labels {
				description += fmt.Sprintf("%s=\"%s\",", sortedVariableLabels[i], label)
			}
			// remove trailing comma + put in brackets:
			description = fmt.Sprintf("{%s}", description[:len(description)-1])
		}
		result += fmt.Sprintf("\n%s%s %v", c.Name, description, lv.value)
	}
	return "\n" + result + "\n"
}

func (c *GaugeDesc) makeCollectionTest(labeledValues ...LV) collectionTest {
	return collectionTest{Name: c.Name, ExpectedResponse: c.expectedCollection(labeledValues...)}
}
