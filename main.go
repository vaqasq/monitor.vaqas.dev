package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {

}

func fetchContainerList() bool {

	containerListAPI := os.Getenv("CONTAINER_LIST_API")
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
