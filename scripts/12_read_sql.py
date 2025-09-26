import sqlite3
import sys
import os

def read_l1vcache_read_misses(db_path):
    """Read L1VCache read-miss values and calculate their average."""
    
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
        
        read_miss_values = []
        
        # Search for L1VCache read-miss entries in each table
        for table in tables:
            table_name = table[0]
            
            # Get column names to identify the right columns
            cursor.execute(f"PRAGMA table_info({table_name});")
            columns = cursor.fetchall()
            column_names = [col[1] for col in columns]
            
            # Check if this table has the required columns
            if 'Location' in column_names and 'What' in column_names and 'Value' in column_names:
                # Find L1VCache read-miss entries
                cursor.execute(f"SELECT Location, Value FROM {table_name} WHERE Location LIKE '%L1VCache%' AND What = 'read-miss'")
                results = cursor.fetchall()
                
                for location, value in results:
                    try:
                        # Convert value to float and add to our list
                        read_miss_values.append(float(value))
                        # print(f"Found read-miss: {location}, Value: {value}")
                    except (ValueError, TypeError):
                        print(f"Warning: Could not convert value '{value}' to float")
        
        # Calculate average
        if read_miss_values:
            average = sum(read_miss_values) / len(read_miss_values)
            print("\n=== Results ===")
            print(f"Found {len(read_miss_values)} L1VCache read-miss values")
            print(f"Average L1VCache read-miss: {average:.2f}")
        else:
            print("\nNo L1VCache read-miss values found in the database.")
                
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
    
    read_l1vcache_read_misses(db_path)