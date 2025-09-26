import sqlite3
import sys
import os
from collections import defaultdict

def read_cache_stats(db_path):
    """Read cache statistics and calculate averages for multiple metrics."""
    
    # Check if file exists
    if not os.path.exists(db_path):
        print(f"Error: Database file '{db_path}' not found.")
        return
    
    # Connect to the database
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Get list of all tables
        cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
        tables = cursor.fetchall()
        
        if not tables:
            print("No tables found in the database.")
            return
        
        # Create a dictionary to store all statistics
        stats = defaultdict(lambda: defaultdict(list))
        
        # Metrics to track
        metrics = ['read-hit', 'read-mshr-hit', 'write-hit', 'write-mshr-hit']
        
        # Cache types to track
        cache_types = ['L1VCache', 'L2Cache']
        
        # Process tables
        for table in tables:
            table_name = table[0]
            
            # Get column names to identify the right columns
            cursor.execute(f"PRAGMA table_info({table_name});")
            columns = cursor.fetchall()
            column_names = [col[1] for col in columns]
            
            # Check if this table has the required columns
            if 'Location' in column_names and 'What' in column_names and 'Value' in column_names:
                # Query for all relevant cache stats
                for cache_type in cache_types:
                    query = f"""
                    SELECT Location, What, Value 
                    FROM {table_name} 
                    WHERE Location LIKE '%{cache_type}%' 
                    AND What IN ('read-hit', 'read-mshr-hit', 'write-hit', 'write-mshr-hit')
                    """
                    cursor.execute(query)
                    results = cursor.fetchall()
                    
                    for location, metric, value in results:
                        try:
                            # Convert value to float and add to our dictionary
                            stats[cache_type][metric].append(float(value))
                        except (ValueError, TypeError):
                            print(f"Warning: Could not convert value '{value}' to float for {location}, {metric}")
        
        # Print results
        print("\n=== Cache Statistics ===")
        
        for cache_type in cache_types:
            print(f"\n{cache_type} Statistics:")
            
            if not stats[cache_type]:
                print(f"  No statistics found for {cache_type}")
                continue
                
            for metric in metrics:
                values = stats[cache_type][metric]
                if values:
                    average = sum(values) / len(values)
                    print(f"  {metric}: {average:.4f} (from {len(values)} samples)")
                else:
                    print(f"  {metric}: No data found")
        
        # Calculate overall averages across all cache types
        print("\nOverall Cache Statistics:")
        for metric in metrics:
            all_values = []
            for cache_type in cache_types:
                all_values.extend(stats[cache_type][metric])
            
            if all_values:
                overall_avg = sum(all_values) / len(all_values)
                print(f"  {metric}: {overall_avg:.4f} (from {len(all_values)} samples)")
            else:
                print(f"  {metric}: No data found")
                
    except sqlite3.Error as e:
        print(f"SQLite error: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    if len(sys.argv) > 1:
        db_path = sys.argv[1]
    else:
        db_path = input("Enter the path to the SQLite database file: ")
    
    read_cache_stats(db_path)