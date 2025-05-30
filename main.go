package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	//"path/filepath"
	"strconv"
	"strings"
	"time"

	"cpu_monitor/alert"

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

	// influxdb config 
	url := os.Getenv("INFLUXDB_URL")
	token := os.Getenv("INFLUXDB_TOKEN")
	org := os.Getenv("INFLUXDB_ORG")
	bucket := os.Getenv("INFLUXDB_BUCKET")

	// alert config
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	alertEmail := os.Getenv("ALERT_EMAIL")
	cpuThreshold, _ := strconv.ParseFloat(os.Getenv("CPU_THRESHOLD"), 64)
	csvFilePath := os.Getenv("CSV_FILE_PATH")

	if url == "" || token == "" || org == "" || bucket == "" {
		log.Fatal("Missing required InfluxDB environment variables. Please check your .env file.")
	}
	if smtpHost == "" || smtpPort == "" || smtpUsername == "" || smtpPassword == "" || alertEmail == "" || cpuThreshold == 0 {
		log.Println("SMTP or threshold not configured; email alerts disabled.")
		smtpHost = ""
	}
	if csvFilePath == "" {
		log.Println("CSV file path not configured; ML predictions disabled.")
		csvFilePath = ""
	}

	// khởi tạo client
	client := influxdb2.NewClient(url, token)
	defer client.Close()

	// write API
	writeAPI := client.WriteAPIBlocking(org, bucket)

	// query API
	queryAPI := client.QueryAPI(org)

	// khởi tạo alert system
	alertSystem := alert.NewAlertSystem(smtpHost, smtpPort, smtpUsername, smtpPassword, alertEmail, cpuThreshold)

	// thu thập mức sdung CPU sau mỗi 10s
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			percent, err := cpu.Percent(0, false)
			if err != nil {
				log.Printf("Error getting CPU usage: %v", err)
				continue
			}

			// tạo một điểm mốc cho influxdb
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

			// write
			if err := writeAPI.WritePoint(context.Background(), point); err != nil {
				log.Printf("Error writing to InfluxDB: %v", err)
			} else {
				log.Printf("Wrote CPU usage: %.2f%%", percent[0])
			}

			// check nếu mức sdung CPU cao và gửi tbao nếu cần
			if smtpHost != "" {
				if err := alertSystem.CheckAndSendAlert(percent[0]); err != nil {
					log.Printf("Error in alert system: %v", err)
				}
			}
		}
	}()

	// truy vấn data trong vòng 10m trở lại và chạy ML sau mỗi 30s
	queryTicker := time.NewTicker(30 * time.Second)
	defer queryTicker.Stop()

	for range queryTicker.C {
		// query
		query := fmt.Sprintf(`from(bucket:"%s")
			|> range(start: -10m)
			|> filter(fn: (r) => r._measurement == "cpu_usage")
			|> filter(fn: (r) => r._field == "percent")
			|> filter(fn: (r) => r.host == "localhost")
			|> aggregateWindow(every: 30s, fn: mean, createEmpty: false)
			|> yield(name: "mean")`, bucket)

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

		// refresh file csv với data mới nhất
		if csvFilePath != "" {
			err := refreshCSV(csvFilePath, query)
			if err != nil {
				log.Printf("Error refreshing CSV: %v", err)
			} else {
				log.Printf("Refreshed %s with latest data", csvFilePath)
				if csvContent, err := os.ReadFile(csvFilePath); err == nil {
					lines := strings.Split(string(csvContent), "\n")
					if len(lines) > 0 {
						log.Printf("CSV header: %s", lines[0])
					}
				}
			}
		}

		// chạy ML nếu đc enable
		if csvFilePath != "" {
			prediction, err := runPythonPredictor(csvFilePath)
			if err != nil {
				log.Printf("Error making ML prediction: %v", err)
			} else {
				fmt.Printf("Predicted CPU usage (next 10s): %.2f%%\n", prediction)
			}
		}
	}
}

// chạy lệnh truy vấn influx để update file csv
func refreshCSV(csvFilePath, query string) error {
	tmpFile, err := os.CreateTemp("", "influx_query_*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp query file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) //clean up


	if _, err := tmpFile.WriteString(query); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write query to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	cmd := exec.Command("influx", "query", "-f", tmpFile.Name(), "--raw")
	log.Printf("Executing command: influx query -f %s --raw", tmpFile.Name())
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run influx query: %v, stderr: %s", err, stderr.String())
	}

	// viết output ra csvFilePath
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %v", csvFilePath, err)
	}
	defer csvFile.Close()

	if _, err := csvFile.Write(out.Bytes()); err != nil {
		return fmt.Errorf("failed to write to CSV file %s: %v", csvFilePath, err)
	}

	return nil
}

// chạy script predict.py và trả về kết quả dự đoán
func runPythonPredictor(csvFilePath string) (float64, error) {
	cmd := exec.Command("python", "ml/predict.py", csvFilePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to run Python script: %v, output: %s", err, out.String())
	}

	output := strings.TrimSpace(out.String())
	prediction, err := strconv.ParseFloat(output, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse Python script output: %v", err)
	}

	return prediction, nil
}