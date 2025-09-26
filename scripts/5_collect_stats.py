import sqlite3
import os
import re
from collections import defaultdict

def generalize_location(loc):
    patterns = [
        (r'GPU\[\d+\]\.CommandProcessor', 'GPU.CommandProcessor'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.CU\[\d+\]', 'GPU.SA.CU'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1VCache\[\d+\]', 'GPU.SA.L1VCache'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1SCache', 'GPU.SA.L1SCache'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1ICache', 'GPU.SA.L1ICache'),
        (r'GPU\[\d+\]\.L2Cache\[\d+\]', 'GPU.L2Cache'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1VCache', 'GPU.SA.L1VCache'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1VTLB\[\d+\]', 'GPU.SA.L1VTLB'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1STLB', 'GPU.SA.L1STLB'),
        (r'GPU\[\d+\]\.SA\[\d+\]\.L1ITLB', 'GPU.SA.L1ITLB'),
        (r'GPU\[\d+\]\.DRAM\[\d+\]', 'GPU.DRAM'),
        (r'GPU\[\d+\]\.L2ToDRAM\[\d+\]', 'GPU.L2ToDRAM'),
        (r'GPU\[\d+\]\.L2TLB', 'GPU.L2TLB'),
        (r'GPU\[\d+\]\.RDMA', 'GPU.RDMA'),
        (r'GPU\[\d+\]\.L2ToDRAM', 'GPU.L2ToDRAM'),
    ]
    for pat, repl in patterns:
        if re.fullmatch(pat, loc):
            return repl
    return loc

def process_sqlite_file(db_file, output_path):
    conn = sqlite3.connect(db_file)
    cursor = conn.cursor()
    cursor.execute("SELECT Location, What, Value, Unit FROM mgpusim_metrics")
    rows = cursor.fetchall()
    grouped = defaultdict(list)
    for loc, what, value, unit in rows:
        gen_loc = generalize_location(loc)
        grouped[(gen_loc, what, unit)].append(value)
    with open(output_path, 'w') as f:
        for (gen_loc, what, unit), values in grouped.items():
            # Filter out None values
            filtered_values = [v for v in values if v is not None]
            if filtered_values:
                avg_value = sum(filtered_values) / len(filtered_values)
                f.write(f"{gen_loc}, {what}, {avg_value:.10f}, {unit}\n")
    cursor.close()
    conn.close()

benchmarks = [
    ('normal', 'fir'),
    ('l1-prefetcher', 'fir'),
    ('normal', 'matrixmultiplication'),
    ('l1-prefetcher', 'matrixmultiplication'),
    ('normal', 'conv2d'),
    ('l1-prefetcher', 'conv2d')
]

for bench_dir, bench_name in benchmarks:
    dir_path = os.path.join(bench_dir, bench_name)
    if not os.path.isdir(dir_path):
        continue
    sqlite_file = None
    for fname in os.listdir(dir_path):
        if fname.endswith('.sqlite3'):
            sqlite_file = os.path.join(dir_path, fname)
            break
    if sqlite_file:
        output_path = os.path.join(dir_path, 'stats.log')
        process_sqlite_file(sqlite_file, output_path)
        print(f"Wrote stats to {output_path}")
    else:
        print(f"No .sqlite3 file found in {dir_path}")
