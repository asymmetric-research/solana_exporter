package main

import (
	"k8s.io/klog/v2"
)

func assertf(condition bool, format string, args ...any) {
	if !condition {
		klog.Fatalf(format, args...)
	}
}
