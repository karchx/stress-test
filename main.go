package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	log "github.com/gothew/l-og"
)

const (
	TotalRequests    = 1000     // Total de peticiones a realizar
	MaxConcurrent    = 100      // Máximo de peticiones concurrentes
	RequestTimeout   = 10       // Timeout en segundos para cada petición
	RateLimit        = 50       // Peticiones por segundo
)

var url string = ""

type RequestResult struct {
	URL string
	Duration time.Duration
	StatusCode int
	TimeStamp     string
}

func random(min, max int) int {
	return rand.Intn(max - min) + min
}

var client = &http.Client{
	Timeout: time.Second * RequestTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     30 * time.Second,
	},
}

func makeRequest(id string, resultChan chan<- RequestResult, rateLimiter <-chan time.Time) {
	<-rateLimiter

	fullUrl := url + "/" + id
	startTime := time.Now()

	result := RequestResult{
		URL: fullUrl,
		TimeStamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	resp, err := client.Get(fullUrl)
	if err != nil {
		result.StatusCode = 0
		result.Duration = time.Since(startTime)
		log.Error("Error en la peticion: ", err)
	} else {
		defer resp.Body.Close()
		result.StatusCode = resp.StatusCode
		result.Duration = time.Since(startTime)
	}

	log.Infof("URL: %s - Tiempo de respuesta: %v - Status %d", result.URL, result.Duration, result.StatusCode)

	resultChan <- result
}

func writeToCSV(results []RequestResult) error {
	file, err := os.Create(fmt.Sprintf("stress_test_results_%s.csv", 
		time.Now().Format("20060102_150405")))
	if err != nil {
		return fmt.Errorf("error creando archivo CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Timestamp", "URL", "Duration (ms)", "Status Code"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("error escribiendo encabezados: %v", err)
	}

	for _, result := range results {
		row := []string{
			result.TimeStamp,
			result.URL,
			fmt.Sprintf("%.2f", float64(result.Duration.Milliseconds())),
			fmt.Sprintf("%d", result.StatusCode),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error escribiendo fila: %v", err)
		}
	}

	return nil
}

// Estructura para las estadísticas
type StressTestStats struct {
	TotalRequests    int                    `json:"total_requests"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time"`
	TotalDuration    string                 `json:"total_duration"`
	AverageDuration  string                 `json:"average_duration"`
	MinDuration      string                 `json:"min_duration"`
	MaxDuration      string                 `json:"max_duration"`
	TotalErrors      int                    `json:"total_errors"`
	StatusCodeCounts map[int]int            `json:"status_code_counts"`
	Configuration    map[string]interface{} `json:"configuration"`
}

func generateStats(results []RequestResult, startTime time.Time) *StressTestStats {
	endTime := time.Now()
	var totalDuration time.Duration
	statusCodes := make(map[int]int)
	var minDuration, maxDuration time.Duration
	errors := 0

	if len(results) > 0 {
		minDuration = results[0].Duration
		maxDuration = results[0].Duration
	}

	for _, result := range results {
		totalDuration += result.Duration
		statusCodes[result.StatusCode]++
		
		if result.Duration < minDuration {
			minDuration = result.Duration
		}
		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}
		
	}

	avgDuration := totalDuration / time.Duration(len(results))

	stats := &StressTestStats{
		TotalRequests:    len(results),
		StartTime:        startTime,
		EndTime:          endTime,
		TotalDuration:    totalDuration.String(),
		AverageDuration:  avgDuration.String(),
		MinDuration:      minDuration.String(),
		MaxDuration:      maxDuration.String(),
		TotalErrors:      errors,
		StatusCodeCounts: statusCodes,
		Configuration: map[string]interface{}{
			"total_requests":     TotalRequests,
			"max_concurrent":     MaxConcurrent,
			"request_timeout":    RequestTimeout,
			"rate_limit":        RateLimit,
			"target_url":        url,
		},
	}
	return stats
}

func writeStatsToJSON(stats *StressTestStats) error {
	filename := fmt.Sprintf("stress_test_stats_%s.json", 
		time.Now().Format("20060102_150405"))
	
	jsonData, err := json.MarshalIndent(stats, "", "    ")
	if err != nil {
		return fmt.Errorf("error codificando JSON: %v", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error escribiendo archivo JSON: %v", err)
	}

	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var ids = []string{
		"67103f408600ede084389b62",
		"671024c08600ede084382347",
		"6712b57c8600ede0843fa7dd",
		"6712b5df8600ede0843fa840",
		"6712b6148600ede0843fa8ab",
	}
	resultsChan := make(chan RequestResult, TotalRequests)
	rateLimiter := time.Tick(time.Second / time.Duration(RateLimit))

	// Semáforo para limitar la concurrencia máxima
	sem := make(chan struct{}, MaxConcurrent)

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < TotalRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-sem }()

			index  := random(0, len(ids) -1 )
			makeRequest(ids[index], resultsChan, rateLimiter)
		}(i)
	}

	var results []RequestResult

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)	
	}

	stats := generateStats(results, startTime)
	if err := writeStatsToJSON(stats); err != nil {
		log.Error("Error guardando estadísticas:", err)
	}

	if err := writeToCSV(results); err != nil {
		log.Error("Error escribiendo CSV:", err)
	} else {
		log.Info("Resultados guardados en CSV exitosamente")
	}

}