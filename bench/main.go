package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/davidroman0O/comfylite3"
)

type BenchmarkResult struct {
	TotalInserts      int
	DurationSeconds   float64
	InsertsPerSecond  float64
	SuccessfulInserts int64
	FailedInserts     int64
}

func runBenchmark(iterations int, duration time.Duration) (*BenchmarkResult, error) {
	// Create a new ComfyDB instance
	comfy, err := comfylite3.New(
		comfylite3.WithMemory(),
		comfylite3.WithRetryAttempts(3),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ComfyDB: %v", err)
	}
	defer comfy.Close()

	// Create table
	createID := comfy.New(func(db *sql.DB) (interface{}, error) {
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS benchmark (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		return nil, err
	})
	if err := waitForResult(comfy, createID); err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var successCount, failCount atomic.Int64
	startTime := time.Now()

	// Run benchmark iterations
	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			id := comfy.New(func(db *sql.DB) (interface{}, error) {
				_, err := db.Exec("INSERT INTO benchmark (value) VALUES (?)",
					fmt.Sprintf("test_value_%d", i))
				return nil, err
			})

			result := <-comfy.WaitForChn(id)
			if _, ok := result.(error); ok {
				failCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}
	}

	elapsed := time.Since(startTime)
	elapsedSeconds := elapsed.Seconds()
	insertsPerSecond := float64(iterations) / elapsedSeconds

	return &BenchmarkResult{
		TotalInserts:      iterations,
		DurationSeconds:   elapsedSeconds,
		InsertsPerSecond:  insertsPerSecond,
		SuccessfulInserts: successCount.Load(),
		FailedInserts:     failCount.Load(),
	}, nil
}

func waitForResult(comfy *comfylite3.ComfyDB, id uint64) error {
	result := <-comfy.WaitForChn(id)
	if err, ok := result.(error); ok {
		return err
	}
	return nil
}

func main() {
	const (
		numIterations = 10 // Number of benchmark runs
		insertCount   = 10000
		duration      = 30 * time.Second
	)

	var totalInsertRate float64
	successfulBenchmarks := 0

	fmt.Printf("Running %d benchmark iterations with %d inserts each...\n\n", numIterations, insertCount)

	for i := 0; i < numIterations; i++ {
		fmt.Printf("Benchmark run %d/%d:\n", i+1, numIterations)

		result, err := runBenchmark(insertCount, duration)
		if err != nil {
			log.Printf("Benchmark run %d failed: %v\n", i+1, err)
			continue
		}

		fmt.Printf("  Total inserts: %d\n", result.TotalInserts)
		fmt.Printf("  Duration: %.2f seconds\n", result.DurationSeconds)
		fmt.Printf("  Inserts/second: %.2f\n", result.InsertsPerSecond)
		fmt.Printf("  Successful: %d\n", result.SuccessfulInserts)
		fmt.Printf("  Failed: %d\n\n", result.FailedInserts)

		totalInsertRate += result.InsertsPerSecond
		successfulBenchmarks++
	}

	if successfulBenchmarks > 0 {
		averageInsertRate := totalInsertRate / float64(successfulBenchmarks)
		fmt.Printf("Average inserts/second across %d successful runs: %.2f\n",
			successfulBenchmarks, averageInsertRate)
	} else {
		fmt.Println("No successful benchmark runs completed")
	}
}
