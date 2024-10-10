package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSelectFromSchedule(t *testing.T) {
	selected := SelectFromSchedule(staticLeaderSchedule, 5, 10)
	assert.Equal(t,
		map[string][]int64{"aaa": {6, 9}, "bbb": {7, 10}, "ccc": {5, 8}},
		selected,
	)
}

func TestGetTrimmedLeaderSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schedule, err := GetTrimmedLeaderSchedule(ctx, &staticRPCClient{}, []string{"aaa", "bbb"}, 10, 10)
	assert.NoError(t, err)

	assert.Equal(t, map[string][]int64{"aaa": {10, 13, 16, 19, 22}, "bbb": {11, 14, 17, 20, 23}}, schedule)
}
