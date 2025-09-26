import os
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np
plt.style.use('seaborn-v0_8-deep')

# Label mappings
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
    'demand_average_latency': 'Overall Average',
    'prefetch_average_latency': 'Prefetch Access',
    'remote_average_latency': 'Remote Request'
}

# Normalize function
def normalize(series):
    return series / series.max()

# Conversion function for latency
def seconds_to_nanoseconds(value):
    return value * 1e9

# Benchmarks
benchmarks = [
    "pagerank",
    # "matrixtranspose",
    "floydwarshall",
    # "matrixmultiplication",
    "fir",
    "simpleconvolution"
]

# Collect data for unified graphs
all_reads = {}
all_latencies = {}
all_prefetch_hit_miss = {}
all_mshr_hits = {}

for benchmark in benchmarks:
    # Paths
    normal_file = f"./normal/{benchmark}/metrics.csv"
    prefetch_file = f"./prefetcher/{benchmark}/metrics.csv"

    # Load data
    normal_df = pd.read_csv(normal_file, header=None, names=["ID", "Location", "Category", "Value"])
    prefetch_df = pd.read_csv(prefetch_file, header=None, names=["ID", "Location", "Category", "Value"])

    for df in [normal_df, prefetch_df]:
        df["Category"] = df["Category"].str.strip()
        df["Value"] = pd.to_numeric(df["Value"], errors="coerce")
        df = df[df["Location"].str.contains("L1VCache")]

    # Read metrics
    read_categories = list(read_label_map.keys())
    latency_categories = list(latency_label_map.keys())

    normal_reads = normal_df[normal_df["Category"].isin(read_categories)].groupby("Category")["Value"].mean()
    prefetch_reads = prefetch_df[prefetch_df["Category"].isin(read_categories)].groupby("Category")["Value"].mean()

    normal_latencies = normal_df[normal_df["Category"].isin(latency_categories)].groupby("Category")["Value"].mean().apply(seconds_to_nanoseconds)
    prefetch_latencies = prefetch_df[prefetch_df["Category"].isin(latency_categories)].groupby("Category")["Value"].mean().apply(seconds_to_nanoseconds)

    # Store data
    all_reads[benchmark] = {
        'Demand Reads': [normal_reads['local-demand-read'], normal_reads['remote-demand-read']],
        'Prefetch Reads': [prefetch_reads['local-prefetch-read'], prefetch_reads['remote-prefetch-read']]
    }
    all_latencies[benchmark] = {
        'Demand': normal_latencies.rename(latency_label_map),
        'Prefetch': prefetch_latencies.rename(latency_label_map)
    }
    all_prefetch_hit_miss[benchmark] = [prefetch_reads['read-prefetch-hit'], prefetch_reads['read-prefetch-miss']]
    all_mshr_hits[benchmark] = [normal_reads['read-mshr-hit'], prefetch_reads['read-mshr-hit']]

output_dir = "./graphs/unified"
os.makedirs(output_dir, exist_ok=True)

# 1. Prefetch Read Hit vs Miss Ratio
plt.figure(figsize=(10, 6))
colors = ['#66b3ff', '#ff9999']

for i, benchmark in enumerate(benchmarks):
    total_reads = sum(all_prefetch_hit_miss[benchmark])
    hits = all_prefetch_hit_miss[benchmark][0] / total_reads
    misses = all_prefetch_hit_miss[benchmark][1] / total_reads
    
    # Only add labels for the first iteration
    if i == 0:
        plt.bar(benchmark, hits, color=colors[0], label='Prefetch Hits')
        plt.bar(benchmark, misses, bottom=hits, color=colors[1], label='Prefetch Misses')
    else:
        plt.bar(benchmark, hits, color=colors[0])
        plt.bar(benchmark, misses, bottom=hits, color=colors[1])
    
    plt.text(benchmark, hits / 2, f'{hits * 100:.1f}%', ha='center')
    plt.text(benchmark, 1 - misses / 2, f'{misses * 100:.1f}%', ha='center')

plt.title("Prefetch Read Hits vs Misses for All Benchmarks", fontsize=14, fontweight='bold')
plt.ylabel("Proportion", fontsize=12)
plt.xticks(rotation=45)
plt.legend()  # Add legend after the loop
plt.tight_layout()
plt.savefig(os.path.join(output_dir, "prefetch_hit_vs_miss_all.png"), dpi=300)
plt.close()

# 2. Latency Comparison
# plt.figure(figsize=(12, 7))
# ordered_latency_labels = ['Local Request', 'Overall Average', 'Prefetch Access', 'Remote Request']
# combined_latencies_df = pd.DataFrame.from_dict(
#     {k: pd.concat([normalize(v['Demand'].reindex(ordered_latency_labels)), normalize(v['Prefetch'].reindex(ordered_latency_labels))]) for k, v in all_latencies.items()},
#     orient='index'
# )
# combined_latencies_df.plot(kind='bar', width=0.8)
# plt.title("Latency Comparison for All Benchmarks (Normalized)", fontsize=14, fontweight='bold')
# plt.ylabel("Normalized Latency", fontsize=12)
# plt.xticks(rotation=45)
# plt.tight_layout()
# plt.savefig(os.path.join(output_dir, "latency_comparison_all.png"), dpi=300)
# plt.close()
plt.figure(figsize=(15, 7))
bar_width = 0.2
num_benchmarks = len(benchmarks)
num_categories = 3  # Local, Remote, Overall

# Create positions for each group of bars
indices = np.arange(num_benchmarks)

latency_types = ['Local Request', 'Overall Average', 'Remote Request']
colors = ['#1f77b4', '#ff7f0e', '#d62728']

for i, benchmark in enumerate(benchmarks):
    demand_data = normalize(all_latencies[benchmark]['Demand'].reindex(latency_types))
    base_x = i - bar_width  # Center the group of bars for each benchmark
    
    for j, (latency_type, value) in enumerate(demand_data.items()):
        plt.bar(i + j*bar_width - bar_width, value, bar_width,
                label=latency_type if i == 0 else "", color=colors[j])
        if not np.isnan(value):  # Only add label if value is not NaN
            plt.text(i + j*bar_width - bar_width, value, f'{value:.2f}',
                    ha='center', va='bottom', rotation=45)

plt.title("Latency Comparison by Benchmark (Normalized)", fontsize=14, fontweight='bold')
plt.xlabel("Benchmarks", fontsize=12)
plt.ylabel("Normalized Latency", fontsize=12)
plt.xticks(indices, benchmarks, rotation=45)
plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left')
plt.grid(True, axis='y', linestyle='--', alpha=0.7)
plt.tight_layout()
plt.savefig(os.path.join(output_dir, "latency_comparison_all.png"), dpi=300, bbox_inches='tight')
plt.close()

# 3. MSHR Hits Comparison (Stacked, Normalized to Percentage)
mshr_hits_df = pd.DataFrame.from_dict(all_mshr_hits, orient='index', columns=['Normal', 'Prefetch'])
mshr_hits_percent = mshr_hits_df.div(mshr_hits_df.sum(axis=1), axis=0) * 100
plt.figure(figsize=(10, 6))
ax = mshr_hits_percent.plot(kind='bar', stacked=True, width=0.8)
plt.title("MSHR Hits Comparison for All Benchmarks", fontsize=14, fontweight='bold')
plt.ylabel("Percentage (%)", fontsize=12)
plt.xticks(rotation=45)
for p in ax.patches:
    height = p.get_height()
    if height > 0:
        ax.annotate(f'{height:.1f}%', (p.get_x() + p.get_width() / 2., p.get_y() + height / 2), ha='center', va='center')
plt.tight_layout()
plt.savefig(os.path.join(output_dir, "mshr_hits_comparison_all.png"), dpi=300)
plt.close()

# 4. Improved Local vs Remote Split Comparison (Combined in one graph)
plt.figure(figsize=(12, 7))
bar_width = 0.35
indices = range(len(benchmarks))
demand_colors = ['#1f77b4', '#ff7f0e']
prefetch_colors = ['#66b3ff', '#ff9999']

# Axis Limit
y_limit = 5000

for i, benchmark in enumerate(benchmarks):
    # Demand data
    demand_local = all_reads[benchmark]['Demand Reads'][0]
    demand_remote = all_reads[benchmark]['Demand Reads'][1]

    # Prefetch data
    prefetch_local = all_reads[benchmark]['Prefetch Reads'][0]
    prefetch_remote = all_reads[benchmark]['Prefetch Reads'][1]

    # Create individual bar groups
    bars1 = plt.bar([i - bar_width / 2], demand_local, width=bar_width, color=demand_colors[0], label='Demand Local' if i == 0 else "")
    bars2 = plt.bar([i - bar_width / 2], demand_remote, width=bar_width, bottom=demand_local, color=demand_colors[1], label='Demand Remote' if i == 0 else "")

    bars3 = plt.bar([i + bar_width / 2], prefetch_local, width=bar_width, color=prefetch_colors[0], label='Prefetch Local' if i == 0 else "")
    bars4 = plt.bar([i + bar_width / 2], prefetch_remote, width=bar_width, bottom=prefetch_local, color=prefetch_colors[1], label='Prefetch Remote' if i == 0 else "")

    # Annotate values
    for bars in [bars1, bars2, bars3, bars4]:
        for bar in bars:
            height = bar.get_height()
            y_position = bar.get_y() + height / 2

            # If the bar exceeds y-limit, place value above the bar
            if height > y_limit:
                plt.annotate(f'{int(height)}*', 
                             (bar.get_x() + bar.get_width() / 2., y_limit + 200),  # Position above limit
                             ha='center', va='bottom', fontsize=8, rotation=90)
            else:
                # Place value inside bar if within limit
                plt.annotate(f'{int(height)}', 
                             (bar.get_x() + bar.get_width() / 2., y_position), 
                             ha='center', va='center', fontsize=8)

# Labels and Limits
plt.title("Local vs Remote Split for Demand and Prefetch Reads", fontsize=14, fontweight='bold')
plt.ylabel("Read Count", fontsize=12)
plt.ylim(0, y_limit + 1000)  # Extend the y-limit slightly for labels above bars
plt.xticks(indices, benchmarks, rotation=45)
plt.legend()

plt.tight_layout()
plt.savefig(os.path.join(output_dir, "split_reads_comparison_all.png"), dpi=300)
plt.close()

