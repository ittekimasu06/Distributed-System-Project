# CPU Metrics Monitor

A Go-based application that monitors CPU usage, stores data in InfluxDB, sends email alerts for high usage, and predicts future CPU usage using a machine learning model in Python.

## Features
- **Real-Time CPU Monitoring**: Collects CPU usage every 10 seconds and stores it in InfluxDB.
- **Email Alerts**: Sends email notifications when CPU usage exceeds a threshold (default: 50%) for 30 seconds.
- **ML Predictions**: Uses a linear regression model to predict CPU usage for the next 10 seconds based on the last 10 minutes of data, stored in `query.csv`.
- **Dynamic CSV Refresh**: Updates `query.csv` with the latest InfluxDB data every 30 seconds for ML predictions.

## Prerequisites
- **Go**: Version 1.22 or higher.
- **Python**: Version 3.12 or higher.
- **InfluxDB**: Version 2.x, running locally or accessible.
- **SMTP Server**: Gmail or another SMTP server for email alerts.
- **Git**: For cloning the repository.

## Setup
1. **Clone the Repository**:
   ```bash
   git clone https://github.com/ittekimasu06/Distributed-System-Project.git
   cd cpu_metrics_influxdb
   ```

2. **Install Go Dependencies**:
   ```bash
   go mod tidy
   ```
   Required packages:
   - `github.com/influxdata/influxdb-client-go/v2`
   - `github.com/joho/godotenv`
   - `github.com/shirou/gopsutil/v4/cpu`

3. **Install Python Dependencies**:
   ```bash
   cd ml
   python -m pip install -r requirements.txt --user
   ```
   Required packages:
   - `pandas==2.2.2`
   - `scikit-learn==1.5.2`

4. **Set Up InfluxDB**:
   - Install and run InfluxDB 2.x (e.g., `http://localhost:8086`).
   - Create a bucket (e.g., `cpu_metrics`) and obtain an API token.
   - Configure the InfluxDB CLI:
     ```bash
     influx config create --config-name cpu_monitor --host-url http://localhost:8086 --org your-org --token your-influxdb-token --active
     ```

5. **Create `.env` File**:
   - Copy `.env.example` (if provided) or create `.env` in the project root:
     ```plaintext
     INFLUXDB_URL=http://localhost:8086
     INFLUXDB_TOKEN=your-influxdb-token
     INFLUXDB_ORG=your-org
     INFLUXDB_BUCKET=cpu_metrics
     SMTP_HOST=smtp.gmail.com
     SMTP_PORT=587
     SMTP_USERNAME=your-email@gmail.com
     SMTP_PASSWORD=your-app-specific-password
     ALERT_EMAIL=recipient@example.com
     CPU_THRESHOLD=80.0
     CSV_FILE_PATH=query.csv
     ```
   - For Gmail, use an [App Password](https://support.google.com/accounts/answer/185833).

6. **Verify Setup**:
   - Ensure InfluxDB is running and accessible.
   - Test Python script:
     ```bash
     python ml/predict.py query.csv
     ```

## Usage
1. **Run the Application**:
   ```bash
   go run main.go
   ```
   - Collects CPU usage every 10 seconds.
   - Queries InfluxDB and refreshes `query.csv` every 30 seconds.
   - Sends email alerts for high CPU usage.
   - Outputs ML predictions for the next 10 seconds.

2. **Example Output**:
   ```
   2025-05-31T02:40:10+07:00 Wrote CPU usage: 23.50%
   CPU Usage (Last 10 Minutes):
   Time: 2025-05-31T02:40:45+07:00, CPU Usage: 23.50%
   Time: 2025-05-31T02:40:55+07:00, CPU Usage: 25.30%
   Refreshed query.csv with latest data
   Predicted CPU usage (next 10s): 24.75%
   ```

3. **Simulate High CPU Usage** (for testing alerts):
   - On Windows (PowerShell):
     ```powershell
     while ($true) { $x = 1 }
     ```
   - Check `ALERT_EMAIL` inbox for alerts.

## File Structure
```
cpu_metrics_influxdb/
├── .env              # Environment variables (ignored by .gitignore)
├── .gitignore        # Git ignore file
├── main.go           # Main Go application
├── alert/            # Email alert module
│   └── alert.go
├── ml/               # Machine learning module
│   ├── predict.py    # Python script for CPU usage prediction
│   └── requirements.txt
├── query.csv         # CSV file with InfluxDB data for ML predictions
└── README.md         # This file
```

