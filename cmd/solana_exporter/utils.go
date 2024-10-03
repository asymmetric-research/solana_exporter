package main

import (
	"encoding/json"
	"fmt"
)

func assertf(condition bool, format string, args ...any) {
	if !condition {
		panic(fmt.Errorf(format, args...))
	}
}

// prettyPrint is just a useful debugging function
func prettyPrint(obj any) {
	// For pretty-printed JSON, use json.MarshalIndent
	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling to pretty JSON:", err, ". obj: ", obj)
		return
	}

	// Print the pretty JSON string
	fmt.Println(string(prettyJSON))
}
