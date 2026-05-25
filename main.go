package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
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

	ContainerInfoList, IDs := fetchContainerList(httpc)
	fmt.Printf("\nContainer List Data\n%v\n", ContainerInfoList)
	fmt.Printf("Live Container Metrics\n%v\n", fetchLiveContainerMetrics(httpc, IDs)) // This is, by default, a live stream, but I made it just 1 json"

}

func fetchContainerList(client http.Client) (string, []string) {

	/* Nested structs to recieve json data*/
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

	/* Debugging code, Raw Json
	body, _ := io.ReadAll(response.Body)
	return string(body)
	*/

	/* Unpacking json */

	var resp []Resp
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		fmt.Printf("Error while unpacking json for ContainerList: %v", err)
	}
	// Responses are now in resp.State, resp.Status, resp.Health, etc

	// return values
	var sb strings.Builder
	var containerIDs []string

	// Must iterate because Resp is now a slice
	for _, container := range resp {

		// join the slice of strings of Names
		names := strings.Join(container.Names, " ")

		sb.WriteString(fmt.Sprintf("IDs: %s, Names: %s, State: %s, Status: %s, Health.Status: %s, Health.FailingStreak: %d\n",
			container.ID, names, container.State, container.Status, container.Health.Status, container.Health.FailingStreak))

		containerIDs = append(containerIDs, container.ID)
	}

	return sb.String(), containerIDs
}

func fetchLiveContainerMetrics(client http.Client, containerIDs []string) string {

	/* Structs to recieve json data*/

	type CPUUsage struct {
		TotalUsage int `json:"total_usage"`
	}

	type MemoryStats struct {
		Usage int `json:"usage"`
		Limit int `json:"limit"`
	}

	type PreCPUStats struct {
		CPUUsage       CPUUsage `json:"cpu_usage"`
		SystemCPUUsage int      `json:"system_cpu_usage"`
	}

	type CPUStats struct {
		CPUUsage       CPUUsage `json:"cpu_usage"`
		OnlineCPUs     int      `json:"online_cpus"`
		SystemCPUUsage int      `json:"system_cpu_usage"`
	}

	type Resp struct {
		MemoryStats MemoryStats `json:"memory_stats"`
		PreCPUStats PreCPUStats `json:"precpu_stats"`
		CPUStats    CPUStats    `json:"cpu_stats"`
	}

	/* Request json */

	var sb strings.Builder

	for _, ID := range containerIDs {
		response, err := client.Get("http://localhost/containers/" + ID + "/stats?stream=false") //will need to revert this at some point

		if err != nil {
			fmt.Printf("Error while fetching Live Container Metrics: %v", err)
		}

		/* Debugging code
		body, _ := io.ReadAll(response.Body)
		return string(body) */

		/* Unpacking json */

		var resp Resp

		if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
			fmt.Printf("Error while unpacking json for LiveContainerMetrics: %v", err)
		}

		response.Body.Close()

		// Memory usage % = (used_memory / available_memory) * 100.0
		// CPU usage % = (cpu_delta / system_cpu_delta) * number_cpus * 100.0

		memoryUsage := (float64(resp.MemoryStats.Usage) / float64(resp.MemoryStats.Limit)) * 100.0

		cpuDelta := resp.CPUStats.CPUUsage.TotalUsage - resp.PreCPUStats.CPUUsage.TotalUsage
		systemCPUDelta := resp.CPUStats.SystemCPUUsage - resp.PreCPUStats.SystemCPUUsage

		cpuUsage := (float64(cpuDelta) / float64(systemCPUDelta)) * float64(resp.CPUStats.OnlineCPUs) * 100.0

		sb.WriteString(fmt.Sprintf("ID: %s, \nMemory Usage: %.2f%%, CPU Usage: %.2f%%\n", ID, memoryUsage, cpuUsage))

	}
	return sb.String()
}
