package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

func main() {

	/* Creates an internet unix socket in the machine, dial is wrapped in transport, which is wrapped in client */
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

	fmt.Printf("\nContainer List Data\n %v\n", fetchContainerList(httpc))
	fmt.Printf("Live Container Metrics\n %v\n", fetchLiveContainerMetrics(httpc)) // This is, by default, a live stream, but I made it just 1 json"

}

func fetchContainerList(client http.Client) string {

	/* Structs to recieve json data*/
	type Health struct {
		Status        string `json:"Status"`
		FailingStreak int    `json:"FailingStreak"`
	}

	type Resp struct {
		ID     string   `json:"Id"`
		Names  []string `json:"Names"`
		State  string   `json:"State"`
		Status string   `json:"Status"`
		Health Health   `json:"Health"`
	}

	/* Request json */

	response, err := client.Get("http://localhost/containers/json")

	if err != nil {
		fmt.Printf("Error while fetching Container List: %v ", err)
	}

	defer response.Body.Close()

	/* Unpacking json */

	var resp []Resp
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		fmt.Printf("Error while unpacking json for ContainerList: %v", err)
	}
	// Responses are now in resp.State, resp.Status, resp.Health, etc

	body, _ := io.ReadAll(response.Body)
	return string(body)

	/*
		// join the slice of strings of Names
		names := strings.Join(resp.Names, " ")
		containerList := "IDs: " + resp.ID + ", Names: " + names + ", State: " + resp.State +
			", Status: " + resp.Status + ", Health.Status: " + resp.Health.Status + ", Health.FailingStreak: " +
			strconv.Itoa(resp.Health.FailingStreak)

		return containerList
	*/
}

func fetchLiveContainerMetrics(client http.Client) string {

	/* Structs to recieve json data*/

	type Resp struct {
		UsedMemory      int `json:"used_memory"`
		AvailableMemory int `json:"available_memory"`
		CPUDelta        int `json:"cpu_delta"`
		SystemCPUDelta  int `json:"system_cpu_delta"`
		NumberCPUs      int `json:"number_cpus"`
	}

	/* Request json */

	response, err := client.Get("http://localhost/containers/my-website/stats?stream=false")

	if err != nil {
		fmt.Printf("Error while fetching Live Container Metrics: %v", err)
	}

	defer response.Body.Close()

	/* Unpacking json */

	var resp Resp
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		fmt.Printf("Error while unpacking json for LiveContainerMetrics: %v", err)
	}

	// Memory usage % = (used_memory / available_memory) * 100.0
	// CPU usage % = (cpu_delta / system_cpu_delta) * number_cpus * 100.0

	body, _ := io.ReadAll(response.Body)
	return string(body)

	/*
		memoryUsage := (float64(resp.UsedMemory) / float64(resp.AvailableMemory)) * 100.0
		cpuUsage := (float64(resp.CPUDelta) / float64(resp.SystemCPUDelta)) * float64(resp.NumberCPUs) * 100.0

		memoryUsageString := strconv.FormatFloat(float64(memoryUsage), 'f', 2, 64)
		cpuUsageString := strconv.FormatFloat(float64(cpuUsage), 'f', 2, 64)

		containerStats := "Memory Usage: " + memoryUsageString + ", CPU Usage: " + cpuUsageString

		return containerStats */

}
