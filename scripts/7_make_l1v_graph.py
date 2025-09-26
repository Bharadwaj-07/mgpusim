import os
import pandas as pd
import matplotlib.pyplot as plt

# Define the benchmarks
benchmarks = [
    'bitonicsort',
    'fir',
    'floydwarshall',
    'kmeans',
    'matrixmultiplication',
    'matrixtranspose',
    'pagerank',
    'simpleconvolution',
    'spmv',
    'stencil2d',
]

# Define the directories
normal_dir = 'normal/samples'
prefetcher_dir = 'prefetcher/samples'

# Function to read and filter CSV data
def read_and_filter_csv(file_path):
    df = pd.read_csv(file_path, header=None)
    # Filter rows containing 'L1VCache' and 'read-hit' or 'read-miss'
    filtered_df = df[(df[1].str.contains('L1VCache')) & (df[2].isin(['read-hit', 'read-miss']))]
    return filtered_df

# Loop through each benchmark
for benchmark in benchmarks:
    # Construct file paths
    normal_file = os.path.join(normal_dir, f'{benchmark}/metrics.csv')
    prefetcher_file = os.path.join(prefetcher_dir, f'{benchmark}/prefetcher.csv')

    # Read and filter data from normal context
    if os.path.exists(normal_file):
        normal_df = read_and_filter_csv(normal_file)
    else:
        print(f"Normal file not found for benchmark: {benchmark}")
        continue

    # Read and filter data from prefetcher context
    if os.path.exists(prefetcher_file):
        prefetcher_df = read_and_filter_csv(prefetcher_file)
    else:
        print(f"Prefetcher file not found for benchmark: {benchmark}")
        continue

    # Prepare data for plotting
    normal_hits = normal_df[normal_df[2] == 'read-hit'][3].values
    normal_misses = normal_df[normal_df[2] == 'read-miss'][3].values
    prefetcher_hits = prefetcher_df[prefetcher_df[2] == 'read-hit'][3].values
    prefetcher_misses = prefetcher_df[prefetcher_df[2] == 'read-miss'][3].values

    # Create a bar plot for the current benchmark
    x = range(max(len(normal_hits), len(normal_misses), len(prefetcher_hits), len(prefetcher_misses)))

    plt.figure(figsize=(10, 5))
    plt.bar(x, normal_hits, width=0.4, label='Normal Read Hits', color='b', align='center')
    plt.bar([p + 0.4 for p in x], normal_misses, width=0.4, label='Normal Read Misses', color='c', align='center')
    plt.bar([p + 0.8 for p in x], prefetcher_hits, width=0.4, label='Prefetcher Read Hits', color='r', align='center')
    plt.bar([p + 1.2 for p in x], prefetcher_misses, width=0.4, label='Prefetcher Read Misses', color='m', align='center')

    plt.xlabel('L1VCache Instances')
    plt.ylabel('Count')
    plt.title(f'L1VCache Read Hits and Misses for {benchmark}')
    plt.xticks([p + 0.6 for p in x], range(max(len(normal_hits), len(normal_misses), len(prefetcher_hits), len(prefetcher_misses))))
    plt.legend()
    plt.tight_layout()
    plt.show()
