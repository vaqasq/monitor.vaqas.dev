package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Package level vars
type Container struct {
	id                  string
	name                string
	state               string
	status              string
	healthStatus        string
	healthFailingStreak int
}

type ContainerMetrics struct {
	id          string
	memoryUsage float64
	cpuUsage    float64
}

func main() {

	/* Creates an internet unix socket in the machine, dial is wrapped in transport, which is wrapped in client */

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

	// SQLite

	// Open/Make the table
	db, err := sql.Open("sqlite", "history.sqlite")
	if err != nil {
		log.Fatal("Failed to open connection to history.sqlite: ", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	// Create table format
	createTable := `CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		container_id TEXT NOT NULL,
		container_name TEXT NOT NULL,
		status TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		cpu_percent REAL NOT NULL,
		memory_percent REAL NOT NULL
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Error creating table: ", err)
	}

	// Adding data to table every 30 seconds. Needs its own goroutine, running concurrently with main goroutine since it is a ticker.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		insertSQLData(httpc, db)
		for range ticker.C {
			insertSQLData(httpc, db)
		}
	}()

	// Insert http
	http.HandleFunc("/", handler(db))
	http.Handle("/static-files/", http.StripPrefix("/static-files/", http.FileServer(http.Dir("./static-files"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(db *sql.DB) http.HandlerFunc { //wrapped the handler func because I need the to do the api calls.
	return func(w http.ResponseWriter, r *http.Request) {

		// Call sqlite and get the data

		query := "SELECT container_name, status, timestamp, cpu_percent, memory_percent FROM history ORDER BY id DESC LIMIT 2"

		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, "Failed to query database", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type dashboardData struct {
			Name        string
			Status      string
			StatusClass string
			Timestamp   string
			CpuUsage    float64
			MemoryUsage float64
		}
		var data []dashboardData

		for rows.Next() {

			var d dashboardData

			if err := rows.Scan(&d.Name, &d.Status, &d.Timestamp, &d.CpuUsage, &d.MemoryUsage); err != nil {
				http.Error(w, "Failed to query database", http.StatusInternalServerError)
				return
			}

			d.StatusClass = statusClass(d.Status)

			data = append(data, d)

		}

		if err = rows.Err(); err != nil {
			http.Error(w, "Error during row interation", http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFiles("static-files/index.html")

		if err != nil {
			http.Error(w, "Failed to parse static files ", http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to execute templates", http.StatusInternalServerError)
			return
		}

	}
}

func insertSQLData(client http.Client, db *sql.DB) {

	// access slice of containers from fetchContainerList.
	containers := fetchContainerList(client)

	// Get IDs to pass into fetchLiveContainerMetrics
	var containerIDs []string
	for _, cont := range containers {
		containerIDs = append(containerIDs, cont.id)
	}

	// access slice of containerMetrics from fetchLiveContainerMetrics
	containerMetrics := fetchLiveContainerMetrics(client, containerIDs)

	est, _ := time.LoadLocation("America/New_York")
	now := time.Now().In(est).Format("2006-01-02 15:04:05")

	for i, cont := range containers {

		_, err := db.Exec("INSERT INTO history (container_id, container_name, status, timestamp, cpu_percent, memory_percent) VALUES (?, ?, ?, ?, ?, ?)",
			cont.id, cont.name, cont.status, now, containerMetrics[i].cpuUsage, containerMetrics[i].memoryUsage)

		if err != nil {
			log.Println("Error inserting row: ", err)
		}

	}

}

func fetchContainerList(client http.Client) []Container {

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

	// return value
	var containers []Container

	// Must iterate because Resp is now a slice
	for _, container := range resp {

		name := strings.Join(container.Names, " ")

		cont := Container{
			id:                  container.ID,
			name:                name,
			state:               container.State,
			status:              container.Status,
			healthStatus:        container.Health.Status,
			healthFailingStreak: container.Health.FailingStreak,
		}

		containers = append(containers, cont)

	}

	return containers
}

func fetchLiveContainerMetrics(client http.Client, containerIDs []string) []ContainerMetrics {

	type CPUUsage struct {
		TotalUsage int `json:"total_usage"`
	}

	type MemoryStats struct {
		Usage int `json:"usage"`
		Limit int `json:"limit"`
	}

	type CPUStats struct {
		CPUUsage       CPUUsage `json:"cpu_usage"`
		OnlineCPUs     int      `json:"online_cpus"`
		SystemCPUUsage int      `json:"system_cpu_usage"`
	}

	type Resp struct {
		MemoryStats MemoryStats `json:"memory_stats"`
		CPUStats    CPUStats    `json:"cpu_stats"`
	}

	var containerMetrics []ContainerMetrics

	for _, ID := range containerIDs {
		resp1, err := fetchStats(client, ID)
		if err != nil {
			log.Printf("Error fetching first stats for %s: %v", ID, err)
			continue
		}
		var snap1 Resp
		if err := json.NewDecoder(resp1.Body).Decode(&snap1); err != nil {
			resp1.Body.Close()
			log.Printf("Error decoding first stats for %s: %v", ID, err)
			continue
		}
		resp1.Body.Close()

		time.Sleep(100 * time.Millisecond)

		resp2, err := fetchStats(client, ID)
		if err != nil {
			log.Printf("Error fetching second stats for %s: %v", ID, err)
			continue
		}
		var snap2 Resp
		if err := json.NewDecoder(resp2.Body).Decode(&snap2); err != nil {
			resp2.Body.Close()
			log.Printf("Error decoding second stats for %s: %v", ID, err)
			continue
		}
		resp2.Body.Close()

		memoryUsage := (float64(snap2.MemoryStats.Usage) / float64(snap2.MemoryStats.Limit)) * 100.0

		cpuDelta := snap2.CPUStats.CPUUsage.TotalUsage - snap1.CPUStats.CPUUsage.TotalUsage
		systemCPUDelta := snap2.CPUStats.SystemCPUUsage - snap1.CPUStats.SystemCPUUsage

		numCPUs := snap2.CPUStats.OnlineCPUs
		if numCPUs == 0 {
			numCPUs = 1
		}

		var cpuUsage float64
		if systemCPUDelta > 0 {
			cpuUsage = (float64(cpuDelta) / float64(systemCPUDelta)) * float64(numCPUs) * 100.0
		}

		containerMetrics = append(containerMetrics, ContainerMetrics{
			id:          ID,
			memoryUsage: math.Round(memoryUsage*100) / 100,
			cpuUsage:    math.Round(cpuUsage*100) / 100,
		})
	}
	return containerMetrics
}

func fetchStats(client http.Client, id string) (*http.Response, error) {
	return client.Get("http://localhost/containers/" + id + "/stats?stream=false")
}

func statusClass(status string) string {
	s := strings.ToLower(status)
	if strings.Contains(s, "up") {
		return "running"
	}
	if strings.Contains(s, "paused") {
		return "paused"
	}
	return "stopped"
}
