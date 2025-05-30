package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
)

func main() {
	// load file .env
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	//gán các biến môi trường
	url := os.Getenv("INFLUXDB_URL")
	token := os.Getenv("INFLUXDB_TOKEN")
	org := os.Getenv("INFLUXDB_ORG")
	bucket := os.Getenv("INFLUXDB_BUCKET")

	if url == "" || token == "" || org == "" || bucket == "" {
		log.Fatal("Missing required environment variables. Please check your .env file.")
	}

	// khởi tạo influxdb client
	client := influxdb2.NewClient(url, token)
	defer client.Close()

	// write API
	writeAPI := client.WriteAPIBlocking(org, bucket)

	// query API
	queryAPI := client.QueryAPI(org)

	// thu thập mức dùng CPU sau mỗi 10 giây
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			// lấy dữ liệu mức dùng CPU
			percent, err := cpu.Percent(0, false)
			if err != nil {
				log.Printf("Error getting CPU usage: %v", err)
				continue
			}

			// tạo một điểm mốc cho InfluxDB
			point := influxdb2.NewPoint(
				"cpu_usage",
				map[string]string{
					"host": "localhost",
				},
				map[string]interface{}{
					"percent": percent[0],
				},
				time.Now(),
			)

			if err := writeAPI.WritePoint(context.Background(), point); err != nil {
				log.Printf("Error writing to InfluxDB: %v", err)
			} else {
				log.Printf("Wrote CPU usage: %.2f%%", percent[0])
			}
		}
	}()

	// sau mỗi 30s lại truy vấn dữ liệu trong vòng 10 phút trở lại đây
	queryTicker := time.NewTicker(30 * time.Second)
	defer queryTicker.Stop()

	for range queryTicker.C {
		query := fmt.Sprintf(`from(bucket:"%s")
			|> range(start: -10m)
			|> filter(fn: (r) => r._measurement == "cpu_usage")
			|> filter(fn: (r) => r._field == "percent")`, bucket)

		result, err := queryAPI.Query(context.Background(), query)
		if err != nil {
			log.Printf("Error querying InfluxDB: %v", err)
			continue
		}

		fmt.Println("\nCPU Usage (Last 10 Minutes):")
		for result.Next() {
			if result.Err() != nil {
				log.Printf("Error reading query result: %v", result.Err())
				continue
			}
			fmt.Printf("Time: %s, CPU Usage: %.2f%%\n",
				result.Record().Time().Format(time.RFC3339),
				result.Record().Value())
		}
		result.Close()
	}
}
