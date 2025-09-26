#!/usr/bin/env python3
# filepath: /home/pranav/project/summer/prefetcher/mgpusim/amd/scripts/16_compare_read_miss.py

import sqlite3
import os
import sys
import glob

def get_l1vcache_read_misses(benchmark_dir):
    """Read L1VCache read-miss values from SQLite database and calculate their average."""
    
    # Find the SQLite database file (akita_sim_*.sqlite3)
    db_files = glob.glob(os.path.join(benchmark_dir, "akita_sim_*.sqlite3"))
    
    if not db_files:
        return None
    
    db_path = db_files[0]  # Use the first (and usually only) database file
    
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
        tables = cursor.fetchall()
        
        if not tables:
            return None
        
        read_miss_values = []
        
        for table in tables:
            table_name = table[0]
            
            cursor.execute(f"PRAGMA table_info({table_name});")
            columns = cursor.fetchall()
            column_names = [col[1] for col in columns]
            
            if 'Location' in column_names and 'What' in column_names and 'Value' in column_names:
                cursor.execute(f"SELECT Value FROM {table_name} WHERE Location LIKE '%L1VCache%' AND What = 'read-miss'")
                results = cursor.fetchall()
                
                for (value,) in results:
                    try:
                        read_miss_values.append(float(value))
                    except (ValueError, TypeError):
                        continue
        
        if read_miss_values:
            return sum(read_miss_values) / len(read_miss_values)
        else:
            return None
                
    except sqlite3.Error:
        return None
    finally:
        if conn:
            conn.close()

def find_benchmark_folders():
    """Find all benchmark folders that exist in both normal and l1-prefetcher directories."""
    
    base_path = "/home/pranav/project/summer/prefetcher/mgpusim/amd/scripts"
    normal_path = os.path.join(base_path, "normal")
    prefetcher_path = os.path.join(base_path, "l1-prefetcher")
    
    benchmark_folders = set()
    
    # Get benchmarks from normal folder
    if os.path.exists(normal_path) and os.path.isdir(normal_path):
        for item in os.listdir(normal_path):
            item_path = os.path.join(normal_path, item)
            if os.path.isdir(item_path):
                # Check if there's at least one SQLite file
                db_files = glob.glob(os.path.join(item_path, "akita_sim_*.sqlite3"))
                if db_files:
                    benchmark_folders.add(item)
    
    # Get benchmarks from prefetcher folder
    if os.path.exists(prefetcher_path) and os.path.isdir(prefetcher_path):
        for item in os.listdir(prefetcher_path):
            item_path = os.path.join(prefetcher_path, item)
            if os.path.isdir(item_path):
                # Check if there's at least one SQLite file
                db_files = glob.glob(os.path.join(item_path, "akita_sim_*.sqlite3"))
                if db_files:
                    benchmark_folders.add(item)
    
    return sorted(list(benchmark_folders))

def main():
    base_path = "/home/pranav/project/summer/prefetcher/mgpusim/amd/scripts"
    
    # Configuration paths
    configs = {
        "Without Prefetcher": os.path.join(base_path, "normal"),
        "With Prefetcher": os.path.join(base_path, "l1-prefetcher")
    }
    
    # Find all benchmark folders
    benchmark_folders = find_benchmark_folders()
    
    if not benchmark_folders:
        print("No benchmark folders found!")
        print(f"Checked paths:")
        for config_name, path in configs.items():
            print(f"  {config_name}: {path}")
        return
    
    print(f"Found {len(benchmark_folders)} benchmarks: {', '.join(benchmark_folders)}")
    print()
    
    # Collect data for all benchmarks
    results = {}
    
    for benchmark in benchmark_folders:
        results[benchmark] = {}
        
        for config_name, config_path in configs.items():
            benchmark_dir = os.path.join(config_path, benchmark)
            read_miss_avg = get_l1vcache_read_misses(benchmark_dir)
            results[benchmark][config_name] = read_miss_avg
            
            # Debug: Print which files were found/not found
            if read_miss_avg is None:
                db_files = glob.glob(os.path.join(benchmark_dir, "akita_sim_*.sqlite3"))
                if not db_files:
                    print(f"Missing SQLite file in: {benchmark_dir}")
                else:
                    print(f"No L1VCache read-miss data in: {db_files[0]}")
    
    # Print results table
    print("\n" + "=" * 90)
    print(f"{'Benchmark':<20} {'Without Prefetcher':<20} {'With Prefetcher':<18} {'Improvement':<15}")
    print("=" * 90)
    
    valid_results = []
    
    for benchmark in benchmark_folders:
        without_pref = results[benchmark].get("Without Prefetcher")
        with_pref = results[benchmark].get("With Prefetcher")
        
        # Format the values
        without_str = f"{without_pref:.2f}" if without_pref is not None else "N/A"
        with_str = f"{with_pref:.2f}" if with_pref is not None else "N/A"
        
        # Calculate improvement
        if without_pref is not None and with_pref is not None and without_pref > 0:
            improvement = ((without_pref - with_pref) / without_pref) * 100
            improvement_str = f"{improvement:+.1f}%"
            valid_results.append((benchmark, without_pref, with_pref, improvement))
        else:
            improvement_str = "N/A"
        
        print(f"{benchmark:<20} {without_str:<20} {with_str:<18} {improvement_str:<15}")
    
    print("=" * 90)
    
    # Print summary statistics
    if valid_results:
        improvements = [result[3] for result in valid_results]
        avg_improvement = sum(improvements) / len(improvements)
        
        print(f"\nSummary:")
        print(f"Benchmarks with valid data: {len(valid_results)}")
        print(f"Average improvement: {avg_improvement:+.1f}%")
        
        # Show best and worst performing benchmarks
        best = max(valid_results, key=lambda x: x[3])
        worst = min(valid_results, key=lambda x: x[3])
        
        print(f"Best improvement: {best[0]} ({best[3]:+.1f}%)")
        print(f"Worst improvement: {worst[0]} ({worst[3]:+.1f}%)")
        
        # Show detailed results sorted by improvement
        print(f"\nDetailed Results (sorted by improvement):")
        for benchmark, without, with_pref, improvement in sorted(valid_results, key=lambda x: x[3], reverse=True):
            print(f"  {benchmark}: {without:.2f} → {with_pref:.2f} ({improvement:+.1f}%)")
        
    else:
        print("\nNo valid comparison data found.")
        print("Please check that both 'normal' and 'l1-prefetcher' folders exist with akita_sim_*.sqlite3 files.")
        
        # Show what was found
        print(f"\nDiagnostic information:")
        for config_name, config_path in configs.items():
            if os.path.exists(config_path):
                folders = []
                for item in os.listdir(config_path):
                    item_path = os.path.join(config_path, item)
                    if os.path.isdir(item_path):
                        db_files = glob.glob(os.path.join(item_path, "akita_sim_*.sqlite3"))
                        if db_files:
                            folders.append(f"{item} (✓)")
                        else:
                            folders.append(f"{item} (✗)")
                
                print(f"  {config_name} ({config_path}): {len(folders)} folders")
                if folders:
                    print(f"    Folders: {', '.join(sorted(folders))}")
            else:
                print(f"  {config_name}: Path does not exist - {config_path}")

if __name__ == "__main__":
    main()