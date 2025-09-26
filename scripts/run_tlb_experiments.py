
import os
import subprocess
import re
import pandas as pd
import matplotlib.pyplot as plt

# Define the base directory of the project
PROJECT_ROOT = "/home/PCL_2025/Desktop/MGPUSIM/mgpusim"
SHADERARRAY_BUILDER_PATH = os.path.join(PROJECT_ROOT, "amd/samples/runner/timingconfig/shaderarray/builder.go")
BENCHMARKS_DIR = os.path.join(PROJECT_ROOT, "amd/samples")
SCRIPTS_DIR = os.path.join(PROJECT_ROOT, "scripts")
RESULTS_DIR = os.path.join(PROJECT_ROOT, "tlb_experiment_results")

# Ensure results directory exists
os.makedirs(RESULTS_DIR, exist_ok=True)

# L1V TLB sizes to test (original is 64, then 2x, 3x, 4x)
TLB_SIZES = [64, 128, 192, 256]

# Benchmarks to run (from 3_gen_runners.py)
BENCHMARKS = [
    "aes", "atax", "bfs", "bicg", "fastwalshtransform",
    "fft", "fir", "floydwarshall", "kmeans", "matrixmultiplication",
    "matrixtranspose", "nbody", "relu", "simpleconvolution", "spmv", "stencil2d"
]

def run_command(command, cwd=PROJECT_ROOT):
    """Runs a shell command and returns its output."""
    print(f"Running command: {' '.join(command)} in {cwd}")
    result = subprocess.run(command, cwd=cwd, capture_output=True, text=True, check=True)
    print(result.stdout)
    if result.stderr:
        print(result.stderr)
    return result.stdout

def modify_tlb_size(new_size):
    """Modifies the L1V TLB size in the builder.go file."""
    print(f"Modifying L1V TLB size to {new_size} in {SHADERARRAY_BUILDER_PATH}")
    old_string = "        WithNumWays(64)."
    new_string = f"        WithNumWays({new_size})."
    
    # Read the file content
    with open(SHADERARRAY_BUILDER_PATH, 'r') as f:
        content = f.read()

    # Perform the replacement
    modified_content = content.replace(old_string, new_string)

    # Write the modified content back to the file
    with open(SHADERARRAY_BUILDER_PATH, 'w') as f:
        f.write(modified_content)
    print(f"L1V TLB size modified to {new_size}.")

def compile_benchmarks():
    """Compiles all benchmarks."""
    print(f"Compiling benchmarks...")
    for benchmark in BENCHMARKS:
        benchmark_path = os.path.join(BENCHMARKS_DIR, benchmark)
        print(f"Compiling {benchmark} in {benchmark_path}...")
        run_command(["go", "build"], cwd=benchmark_path)
    print("Benchmarks compiled.")

def run_benchmarks(tlb_size_label):
    """Runs all benchmarks for a given TLB size."""
    print(f"Running benchmarks for TLB size: {tlb_size_label}")
    
    # Create a directory for this TLB size's logs
    tlb_results_dir = os.path.join(RESULTS_DIR, f"tlb_{tlb_size_label}")
    os.makedirs(tlb_results_dir, exist_ok=True)

    for benchmark in BENCHMARKS:
        print(f"Running benchmark: {benchmark}")
        benchmark_path = os.path.join(BENCHMARKS_DIR, benchmark)
        
        # Construct the command to run the benchmark
        # The -report-all flag is crucial for collecting TLB hit rates
        command = [
            os.path.join(benchmark_path, benchmark),
            "-timing",
            "-report-all",
            f"-metric-file-name={os.path.join(tlb_results_dir, benchmark)}"
        ]
        
        try:
            run_command(command, cwd=benchmark_path)
        except subprocess.CalledProcessError as e:
            print(f"Error running benchmark {benchmark}: {e}")
            print(f"Stdout: {e.stdout}")
            print(f"Stderr: {e.stderr}")

def collect_stats():
    """Collects IPC and TLB hit/miss stats from all benchmark runs."""
    print("Collecting statistics...")
    all_results = []

    for tlb_size in TLB_SIZES:
        tlb_results_dir = os.path.join(RESULTS_DIR, f"tlb_{tlb_size}")
        for benchmark in BENCHMARKS:
            metric_file = os.path.join(tlb_results_dir, f"{benchmark}.csv")
            if not os.path.exists(metric_file):
                print(f"Metric file not found for {benchmark} at TLB size {tlb_size}: {metric_file}")
                continue

            df = pd.read_csv(metric_file)
            
            # Extract IPC (Instructions Per Cycle)
            # Assuming IPC is available in the CSV, or can be calculated
            # For now, let's assume 'total_instructions' and 'total_cycles' are available
            # If not, this part needs adjustment based on actual CSV content
            ipc = df[df['Metric'] == 'total_instructions']['Value'].iloc[0] / df[df['Metric'] == 'total_cycles']['Value'].iloc[0] if 'total_instructions' in df['Metric'].values and 'total_cycles' in df['Metric'].values else 0

            # Extract L1V TLB hits and misses
            # This will depend on the exact naming in the CSV.
            # I'll look for metrics like 'L1VTLB_hits' and 'L1VTLB_misses'
            l1v_tlb_hits = df[df['Metric'].str.contains('L1VTLB_hits', na=False)]['Value'].sum()
            l1v_tlb_misses = df[df['Metric'].str.contains('L1VTLB_misses', na=False)]['Value'].sum()
            
            all_results.append({
                'TLB_Size': tlb_size,
                'Benchmark': benchmark,
                'IPC': ipc,
                'L1V_TLB_Hits': l1v_tlb_hits,
                'L1V_TLB_Misses': l1v_tlb_misses
            })
    
    results_df = pd.DataFrame(all_results)
    results_df.to_csv(os.path.join(RESULTS_DIR, "tlb_experiment_summary.csv"), index=False)
    print("Statistics collected and saved to tlb_experiment_summary.csv")
    return results_df

def plot_results(results_df):
    """Generates normalized plots for IPC and L1V TLB hit rate."""
    print("Generating plots...")

    # Normalize IPC
    baseline_ipc = results_df[results_df['TLB_Size'] == TLB_SIZES[0]].set_index('Benchmark')['IPC']
    results_df['Normalized_IPC'] = results_df.apply(lambda row: row['IPC'] / baseline_ipc[row['Benchmark']], axis=1)

    # Calculate L1V TLB Hit Rate and Normalize
    results_df['L1V_TLB_Hit_Rate'] = results_df['L1V_TLB_Hits'] / (results_df['L1V_TLB_Hits'] + results_df['L1V_TLB_Misses'])
    results_df['L1V_TLB_Hit_Rate'].fillna(0, inplace=True) # Handle cases with no hits/misses

    baseline_tlb_hit_rate = results_df[results_df['TLB_Size'] == TLB_SIZES[0]].set_index('Benchmark')['L1V_TLB_Hit_Rate']
    results_df['Normalized_L1V_TLB_Hit_Rate'] = results_df.apply(lambda row: row['L1V_TLB_Hit_Rate'] / baseline_tlb_hit_rate[row['Benchmark']] if baseline_tlb_hit_rate[row['Benchmark']] != 0 else 0, axis=1)

    # Plotting Normalized IPC
    plt.figure(figsize=(12, 6))
    for benchmark in BENCHMARKS:
        subset = results_df[results_df['Benchmark'] == benchmark]
        plt.plot(subset['TLB_Size'], subset['Normalized_IPC'], marker='o', label=benchmark)
    plt.title('Normalized IPC vs. L1V TLB Size')
    plt.xlabel('L1V TLB Size (Ways)')
    plt.ylabel('Normalized IPC (vs. 64 Ways)')
    plt.xticks(TLB_SIZES)
    plt.grid(True)
    plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left')
    plt.tight_layout()
    plt.savefig(os.path.join(RESULTS_DIR, "normalized_ipc_plot.png"))
    plt.close()

    # Plotting Normalized L1V TLB Hit Rate
    plt.figure(figsize=(12, 6))
    for benchmark in BENCHMARKS:
        subset = results_df[results_df['Benchmark'] == benchmark]
        plt.plot(subset['TLB_Size'], subset['Normalized_L1V_TLB_Hit_Rate'], marker='o', label=benchmark)
    plt.title('Normalized L1V TLB Hit Rate vs. L1V TLB Size')
    plt.xlabel('L1V TLB Size (Ways)')
    plt.ylabel('Normalized L1V TLB Hit Rate (vs. 64 Ways)')
    plt.xticks(TLB_SIZES)
    plt.grid(True)
    plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left')
    plt.tight_layout()
    plt.savefig(os.path.join(RESULTS_DIR, "normalized_l1v_tlb_hit_rate_plot.png"))
    plt.close()

    print("Plots generated and saved.")

def main():
    original_content = ""
    with open(SHADERARRAY_BUILDER_PATH, 'r') as f:
        original_content = f.read()

    try:
        for tlb_size in TLB_SIZES:
            modify_tlb_size(tlb_size)
            compile_benchmarks()
            run_benchmarks(tlb_size)
        
        results_df = collect_stats()
        plot_results(results_df)
    finally:
        # Restore original content
        print("Restoring original builder.go file...")
        with open(SHADERARRAY_BUILDER_PATH, 'w') as f:
            f.write(original_content)
        print("Original builder.go restored.")

if __name__ == "__main__":
    main()
