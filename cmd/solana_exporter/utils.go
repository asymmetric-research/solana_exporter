package main

import (
	"fmt"
	"k8s.io/klog/v2"
)

func assertf(condition bool, format string, args ...any) {
	if !condition {
		klog.Fatalf(format, args...)
	}
}

func toString(i int64) string {
	return fmt.Sprintf("%v", i)
}
