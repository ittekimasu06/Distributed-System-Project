import sys
import pandas as pd
from sklearn.linear_model import LinearRegression
import numpy as np

def load_and_predict(csv_file):
    try:
        # đọc file csv
        df = pd.read_csv(csv_file, comment='#')
        
        # kiểm tra các cột cần thiết
        if '_time' not in df.columns or '_value' not in df.columns:
            print("Error: CSV missing required columns (_time, _value)", file=sys.stderr)
            sys.exit(1)
        
        # convert thời gian từ định dạng ISO8601 của influxdb sang datetime
        try:
            df['_time'] = pd.to_datetime(df['_time'], format='ISO8601')
        except ValueError as e:
            print(f"Error parsing timestamps: {str(e)}", file=sys.stderr)
            sys.exit(1)
        
        df['seconds'] = (df['_time'] - df['_time'].min()).dt.total_seconds()
        X = df[['seconds']].values
        y = df['_value'].values
        
        # train model
        model = LinearRegression()
        model.fit(X, y)
        
        # đoán trong 10s sắp tới
        last_time = df['seconds'].max()
        future_time = np.array([[last_time + 10]])
        prediction = model.predict(future_time)
        
        # output
        print(f"{prediction[0]:.2f}")
        
    except Exception as e:
        print(f"Error processing CSV: {str(e)}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: predict.py <csv_file>", file=sys.stderr)
        sys.exit(1)
    
    load_and_predict(sys.argv[1])