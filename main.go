package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {
	fmt.Printf("%t", fetchContainerList())
	fmt.Printf("%t", fetchLiveContainerMetrics())
}

func fetchContainerList() bool {

	containerListAPI := os.Getenv("CONTAINER_LIST_API")

	type Health struct {
		Status        string `json:"Status"`
		FailingStreak int    `json:"FailingStreak"`
	}

	type Resp struct {
		State  string `json:"State"`
		Status string `json:"Status"`
		Health Health `json:"Health"`
	}

	resp, err := http.Get(containerListAPI)

	if err != nil {
		fmt.Printf("Error while retrieving ContainerList: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return true
	}

	return false

}

func fetchLiveContainerMetrics() bool {

	liveStreamMetricsAPI := os.Getenv(("LIVE_STREAM_METRICS_API"))

	// Insert STRUCT for the specific data you want to receive.

	resp, err := http.Get(liveStreamMetricsAPI)

	if err != nil {
		fmt.Printf("Error whiel retrieving LiveContainerMetrics: %v ", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return true
	}

	return false

}
