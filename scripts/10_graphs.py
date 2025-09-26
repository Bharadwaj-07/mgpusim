import os
import pandas as pd
import matplotlib.pyplot as plt
plt.style.use('seaborn-v0_8-deep')

# Updated Label mappings
read_label_map = {
    'demand-read': 'Total Demand Reads',
    'prefetch-read': 'Total Prefetch Reads',
    'remote-demand-read': 'Remote Demand Reads',
    'remote-prefetch-read': 'Remote Prefetch Reads',
    'local-demand-read': 'Local Demand Reads',
    'local-prefetch-read': 'Local Prefetch Reads',
    'read-demand-hit': 'Demand Read Hits',
    'read-prefetch-hit': 'Prefetch Read Hits',
    'read-miss': 'Cache Misses',
    'read-prefetch-miss': 'Prefetch Misses',
    'read-mshr-hit': 'MSHR Hits'
}

latency_label_map = {
    'local_average_latency': 'Local Request',
    'remote_average_latency': 'Remote Request',
    'demand_average_latency': 'Overall Average',
    'prefetch_average_latency': 'Prefetch Access'
}

# Conversion function for latency
def seconds_to_nanoseconds(value):
    return value * 1e9

# Unified analysis function
def analyze_performance(normal_file, prefetch_file, output_dir):
    os.makedirs(output_dir, exist_ok=True)

    # Load and preprocess data
    normal_df = pd.read_csv(normal_file, header=None, names=["ID", "Location", "Category", "Value"])
    prefetch_df = pd.read_csv(prefetch_file, header=None, names=["ID", "Location", "Category", "Value"])
    
    for df in [normal_df, prefetch_df]:
        df["Category"] = df["Category"].str.strip()
        df["Value"] = pd.to_numeric(df["Value"], errors="coerce")
        df = df[df["Location"].str.contains("L1VCache")]

    # Metrics categories
    read_categories = list(read_label_map.keys())
    latency_categories = list(latency_label_map.keys())

    # Compute averages
    normal_reads = normal_df[normal_df["Category"].isin(read_categories)].groupby("Category")["Value"].mean()
    prefetch_reads = prefetch_df[prefetch_df["Category"].isin(read_categories)].groupby("Category")["Value"].mean()

    normal_latencies = normal_df[normal_df["Category"].isin(latency_categories)].groupby("Category")["Value"].mean().apply(seconds_to_nanoseconds)
    prefetch_latencies = prefetch_df[prefetch_df["Category"].isin(latency_categories)].groupby("Category")["Value"].mean().apply(seconds_to_nanoseconds)

    # Prefetch Read Hit vs Miss Ratio
    plt.figure(figsize=(8, 6))
    prefetch_hit_miss = [prefetch_reads['read-prefetch-hit'], prefetch_reads['read-prefetch-miss']]
    plt.pie(prefetch_hit_miss, labels=['Prefetch Read Hits', 'Prefetch Read Misses'], autopct='%1.1f%%', colors=['#66b3ff', '#ff9999'])
    plt.title("Prefetch Read Hits vs Misses", fontsize=14, fontweight='bold')
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, "prefetch_hit_vs_miss_ratio.png"), dpi=300)
    plt.close()

    # Remote/Local Split Graph (Stacked Bar Chart)
    plt.figure(figsize=(12, 7))
    split_reads = pd.DataFrame({
        'Demand Reads': [normal_reads['local-demand-read'], normal_reads['remote-demand-read']],
        'Prefetch Reads': [prefetch_reads['local-prefetch-read'], prefetch_reads['remote-prefetch-read']]
    }, index=['Local Reads', 'Remote Reads'])

    split_reads.T.plot(kind='bar', stacked=True, width=0.8, color=['#1f77b4', '#ff7f0e'])
    plt.title("Local vs Remote Split for Demand and Prefetch Reads", fontsize=14, fontweight='bold')
    plt.ylabel("Average Read Count", fontsize=12)
    plt.xlabel("Read Type", fontsize=12)
    plt.xticks(rotation=0)
    plt.legend(title="Read Source")
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, "split_reads_comparison.png"), dpi=300)
    plt.close()

    # Latency Comparison
    plt.figure(figsize=(12, 7))
    combined_latencies = pd.DataFrame({
        'Demand': normal_latencies.rename(latency_label_map),
        'Prefetch': prefetch_latencies.rename(latency_label_map)
    }).dropna()
    combined_latencies.plot(kind='bar', width=0.8)
    plt.title("Latency Comparison", fontsize=14, fontweight='bold')
    plt.ylabel("Latency (nanoseconds)", fontsize=12)
    plt.xlabel("Latency Categories", fontsize=12)
    plt.xticks(rotation=0)
    plt.legend(title="Configuration")
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, "latency_comparison.png"), dpi=300)
    plt.close()

    # MSHR Read Hits Comparison
    plt.figure(figsize=(8, 6))
    mshr_hits = pd.DataFrame({
        'Normal': [normal_reads['read-mshr-hit']],
        'Prefetch': [prefetch_reads['read-mshr-hit']]
    }, index=['MSHR Hits'])
    mshr_hits.plot(kind='bar', width=0.5, color=['#4c72b0', '#dd8452'])
    plt.title("MSHR Read Hits Comparison", fontsize=14, fontweight='bold')
    plt.ylabel("Count", fontsize=12)
    plt.xticks(rotation=0)
    plt.legend(title="Configuration")
    plt.tight_layout()
    plt.savefig(os.path.join(output_dir, "mshr_hits_comparison.png"), dpi=300)
    plt.close()

    # Print metrics
    print("Normal Configuration Metrics:")
    print(normal_reads)
    print(normal_latencies.rename(latency_label_map))

    print("\nPrefetch Configuration Metrics:")
    print(prefetch_reads)
    print(prefetch_latencies.rename(latency_label_map))

# Benchmarks
benchmarks = [
    "pagerank",
    "matrixtranspose",
    "floydwarshall",
    "matrixmultiplication",
    "fir",
    "simpleconvolution",
]

# Analyze all benchmarks
for benchmark in benchmarks:
    normal_input = f"./normal/{benchmark}/metrics.csv"
    prefetch_input = f"./prefetcher/{benchmark}/metrics.csv"
    output_dir = f"./graphs/unified/{benchmark}_analysis"
    analyze_performance(normal_input, prefetch_input, output_dir)