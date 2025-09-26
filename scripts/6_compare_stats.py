import sqlite3
import os
import re
import matplotlib.pyplot as plt


def find_sqlite_files(search_dirs):
    sqlite_files = []
    for search_dir in search_dirs:
        # Only look for .sqlite3 files inside each benchmark subdirectory
        if not os.path.isdir(search_dir):
            continue
        for benchmark in os.listdir(search_dir):
            bench_path = os.path.join(search_dir, benchmark)
            if os.path.isdir(bench_path):
                for file in os.listdir(bench_path):
                    if file.endswith('.sqlite3'):
                        sqlite_files.append(os.path.join(bench_path, file))
    return sqlite_files

def get_metric(conn, what, location_like=None):
    cursor = conn.cursor()
    if location_like:
        cursor.execute("SELECT Value FROM mgpusim_metrics WHERE What = ? AND Location LIKE ?", (what, location_like))
    else:
        cursor.execute("SELECT Value FROM mgpusim_metrics WHERE What = ?", (what,))
    rows = cursor.fetchall()
    values = [float(row[0]) for row in rows]
    return sum(values) / len(values) if values else 0

def get_cache_metrics(conn, cache_type):
    cursor = conn.cursor()
    hit = cursor.execute(f"SELECT Value FROM mgpusim_metrics WHERE What = 'read-hit' AND Location LIKE '%{cache_type}%'").fetchall()
    miss = cursor.execute(f"SELECT Value FROM mgpusim_metrics WHERE What = 'read-miss' AND Location LIKE '%{cache_type}%'").fetchall()
    mshr_hit = cursor.execute(f"SELECT Value FROM mgpusim_metrics WHERE What = 'read-mshr-hit' AND Location LIKE '%{cache_type}%'").fetchall()
    hit_vals = [float(row[0]) for row in hit]
    miss_vals = [float(row[0]) for row in miss]
    mshr_hit_vals = [float(row[0]) for row in mshr_hit]
    total_accesses = sum(hit_vals) + sum(miss_vals) + sum(mshr_hit_vals)
    avg_read_miss = sum(miss_vals) / len(miss_vals) if miss_vals else 0
    miss_rate = (sum(miss_vals) / total_accesses) if total_accesses > 0 else 0
    return avg_read_miss, miss_rate

def get_prefetch_accuracy(benchmark_dir):
    """Read prefetcher statistics from timing_report.txt file."""
    timing_report_path = os.path.join(os.path.dirname(benchmark_dir), "timing_report.txt")
    if not os.path.exists(timing_report_path):
        timing_report_path = os.path.join(benchmark_dir, "timing_report.txt")
        if not os.path.exists(timing_report_path):
            return None
    try:
        with open(timing_report_path, 'r') as f:
            # Read only the last 100 lines
            f.seek(0, os.SEEK_END)
            filesize = f.tell()
            blocksize = 4096
            lines = []
            block = ''
            while len(lines) < 100 and filesize > 0:
                if filesize - blocksize > 0:
                    f.seek(filesize - blocksize)
                else:
                    f.seek(0)
                block = f.read(min(blocksize, filesize)) + block
                lines = block.splitlines()
                filesize -= blocksize
            last_lines = lines[-100:] if len(lines) >= 100 else lines
            content = '\n'.join(last_lines)
            accuracy_matches = re.findall(r'Prefetch Accuracy:\s+(\d+(?:\.\d+)?)%', content)
            if accuracy_matches:
                accuracies = [float(acc) for acc in accuracy_matches]
                avg_accuracy = sum(accuracies) / len(accuracies)
                return avg_accuracy
            else:
                return None
    except IOError:
        return None

def get_runtime_from_report(benchmark_dir):
    """Extract the 'real' runtime from timing_report.txt as seconds."""
    timing_report_path = os.path.join(os.path.dirname(benchmark_dir), "timing_report.txt")
    if not os.path.exists(timing_report_path):
        timing_report_path = os.path.join(benchmark_dir, "timing_report.txt")
        if not os.path.exists(timing_report_path):
            return None
    try:
        with open(timing_report_path, 'r') as f:
            # Read only the last 100 lines
            f.seek(0, os.SEEK_END)
            filesize = f.tell()
            blocksize = 4096
            lines = []
            block = ''
            while len(lines) < 100 and filesize > 0:
                if filesize - blocksize > 0:
                    f.seek(filesize - blocksize)
                else:
                    f.seek(0)
                block = f.read(min(blocksize, filesize)) + block
                lines = block.splitlines()
                filesize -= blocksize
            last_lines = lines[-100:] if len(lines) >= 100 else lines
            content = '\n'.join(last_lines)
            match = re.search(r'real\s+(?:(\d+)m)?(\d+(?:\.\d+)?)s', content)
            if match:
                minutes = int(match.group(1)) if match.group(1) else 0
                seconds = float(match.group(2))
                total_seconds = minutes * 60 + seconds
                return total_seconds
            else:
                return None
    except IOError:
        return None

def print_metrics_for_all_files_and_collect():
    search_dirs = ['normal', 'l1-prefetcher']
    sqlite_files = find_sqlite_files(search_dirs)
    benchmarks_data = []
    for db_path in sqlite_files:
        entry = {}
        entry['benchmark'] = db_path
        try:
            conn = sqlite3.connect(db_path)
            cu_cpi = get_metric(conn, 'cu_CPI')
            cu_inst_count = get_metric(conn, 'cu_inst_count')
            kernel_time = get_metric(conn, 'kernel_time')
            throughput = cu_inst_count / kernel_time if kernel_time else None
            avg_l2_accesses = get_metric(conn, 'read-hit', '%L2Cache%') + \
                              get_metric(conn, 'read-miss', '%L2Cache%') + \
                              get_metric(conn, 'read-mshr-hit', '%L2Cache%')
            avg_l2_accesses = avg_l2_accesses / 3 if avg_l2_accesses else None

            # L1 metrics
            avg_l1_read_miss, l1_miss_rate = get_cache_metrics(conn, 'L1VCache')
            # L2 metrics
            avg_l2_read_miss, l2_miss_rate = get_cache_metrics(conn, 'L2Cache')

            # Runtime from timing_report.txt
            benchmark_dir = os.path.dirname(db_path)
            runtime_seconds = get_runtime_from_report(benchmark_dir)
            if runtime_seconds is not None:
                runtime_min = int(runtime_seconds // 60)
                runtime_sec = runtime_seconds % 60
                runtime_str = f"{runtime_min}m{runtime_sec:.3f}s"
            else:
                runtime_str = "N/A"

            # Prefetcher accuracy if benchmark is from l1-prefetcher
            prefetcher_accuracy = None
            if "l1-prefetcher" in db_path:
                accuracy = get_prefetch_accuracy(benchmark_dir)
                if accuracy is not None:
                    prefetcher_accuracy = accuracy

            # Fill dictionary
            entry['runtime'] = runtime_str
            entry['cu_CPI'] = cu_cpi
            entry['cu_inst_count'] = cu_inst_count
            entry['throughput'] = throughput
            entry['average_L2_accesses'] = avg_l2_accesses
            entry['average_L1_read_miss'] = avg_l1_read_miss
            entry['average_L2_read_miss'] = avg_l2_read_miss
            entry['L1_miss_rate'] = l1_miss_rate
            entry['L2_miss_rate'] = l2_miss_rate
            entry['prefetcher_accuracy'] = prefetcher_accuracy

            benchmarks_data.append(entry)
            conn.close()
        except Exception as e:
            print(f"Error reading {db_path}: {e}")
    return benchmarks_data

def runtime_to_hhmm(runtime_str):
    if runtime_str == "N/A":
        return "N/A"
    match = re.match(r'(?:(\d+)m)?(\d+(?:\.\d+)?)s', runtime_str)
    if match:
        minutes = int(match.group(1)) if match.group(1) else 0
        seconds = float(match.group(2))
        total_minutes = minutes + int(seconds // 60)
        remaining_seconds = seconds % 60
        hours = total_minutes // 60
        mins = total_minutes % 60
        # If seconds > 30, round up the minute
        if remaining_seconds >= 30:
            mins += 1
            if mins == 60:
                hours += 1
                mins = 0
        return f"{int(hours):02d}:{int(mins):02d}"
    return "N/A"

def generate_comparison_table_image(benchmarks_data, output_path="benchmark_comparison_table.png"):
    headers = [
        "Benchmark", "Read Miss (W/O)", "Read Miss (W)", "Miss Rate (W/O)", "Miss Rate (W)",
        "Runtime (HH:MM)", "IPC Change", "Throughput Change", "Prefetch Accuracy",
        "L2 Accesses (W/O)", "L2 Accesses (W)", "L2 Miss Rate (W/O)", "L2 Miss Rate (W)"
    ]
    table_data = []

    paired = {}
    for entry in benchmarks_data:
        parts = entry['benchmark'].split('/')
        if len(parts) >= 2:
            name = parts[1]
            config = 'W' if 'l1-prefetcher' in entry['benchmark'] else 'W/O'
            if name not in paired:
                paired[name] = {}
            paired[name][config] = entry

    def runtime_to_seconds(runtime_str):
        if runtime_str == "N/A":
            return 0
        match = re.match(r'(?:(\d+)m)?(\d+(?:\.\d+)?)s', runtime_str)
        if match:
            minutes = int(match.group(1)) if match.group(1) else 0
            seconds = float(match.group(2))
            return minutes * 60 + seconds
        return 0

    for name, configs in paired.items():
        wo = configs.get('W/O', {})
        w = configs.get('W', {})
        read_miss_wo = f"{wo.get('average_L1_read_miss', 'N/A'):.2f}" if 'average_L1_read_miss' in wo else "N/A"
        read_miss_w = f"{w.get('average_L1_read_miss', 'N/A'):.2f}" if 'average_L1_read_miss' in w else "N/A"
        miss_rate_wo = f"{wo.get('L1_miss_rate', 0) * 100:.2f}%" if 'L1_miss_rate' in wo else "N/A"
        miss_rate_w = f"{w.get('L1_miss_rate', 0) * 100:.2f}%" if 'L1_miss_rate' in w else "N/A"
        runtime_wo = wo.get('runtime', "N/A")
        runtime_w = w.get('runtime', "N/A")
        sec_wo = runtime_to_seconds(runtime_wo)
        sec_w = runtime_to_seconds(runtime_w)
        max_runtime_raw = runtime_wo if sec_wo >= sec_w else runtime_w
        max_runtime = runtime_to_hhmm(max_runtime_raw)
        ipc_wo = 1 / wo['cu_CPI'] if 'cu_CPI' in wo and wo['cu_CPI'] else None
        ipc_w = 1 / w['cu_CPI'] if 'cu_CPI' in w and w['cu_CPI'] else None
        if ipc_wo and ipc_w:
            ipc_improvement = ((ipc_w - ipc_wo) / ipc_wo) * 100
            ipc_improvement_str = f"{ipc_improvement:+.2f}%"
        else:
            ipc_improvement_str = "N/A"
        throughput_wo = wo.get('throughput', None)
        throughput_w = w.get('throughput', None)
        if throughput_wo and throughput_w:
            throughput_improvement = ((throughput_w - throughput_wo) / throughput_wo) * 100
            throughput_improvement_str = f"{throughput_improvement:+.2f}%"
        else:
            throughput_improvement_str = "N/A"
        prefetch_accuracy_str = f"{w.get('prefetcher_accuracy', 'N/A'):.2f}%" if 'prefetcher_accuracy' in w and w.get('prefetcher_accuracy', None) is not None else "N/A"
        l2_accesses_wo = f"{wo.get('average_L2_accesses', 'N/A'):.2f}" if 'average_L2_accesses' in wo else "N/A"
        l2_accesses_w = f"{w.get('average_L2_accesses', 'N/A'):.2f}" if 'average_L2_accesses' in w else "N/A"
        # Multiply L2 miss rates by 100 for percentage
        l2_miss_rate_wo = f"{wo.get('L2_miss_rate', 0) * 100:.2f}%" if 'L2_miss_rate' in wo else "N/A"
        l2_miss_rate_w = f"{w.get('L2_miss_rate', 0) * 100:.2f}%" if 'L2_miss_rate' in w else "N/A"

        table_data.append([
            name, read_miss_wo, read_miss_w, miss_rate_wo, miss_rate_w, max_runtime,
            ipc_improvement_str, throughput_improvement_str, prefetch_accuracy_str,
            l2_accesses_wo, l2_accesses_w, l2_miss_rate_wo, l2_miss_rate_w
        ])

    fig, ax = plt.subplots(figsize=(22, 6))
    ax.axis('off')
    table = ax.table(cellText=table_data, colLabels=headers, cellLoc='center', loc='center')
    table.auto_set_font_size(False)
    table.set_fontsize(10)
    table.scale(1.2, 2.0)

    for i in range(len(headers)):
        table[(0, i)].set_facecolor('#2E4057')
        table[(0, i)].set_text_props(weight='bold', color='white', fontsize=11)
        table[(0, i)].set_height(0.08)

    for i in range(1, len(table_data) + 1):
        for j in range(len(headers)):
            table[(i, j)].set_facecolor('#f8f9fa' if i % 2 == 0 else '#ffffff')
            table[(i, j)].set_height(0.07)
            table[(i, j)].set_text_props(fontsize=10)

    plt.title('Benchmark Read Miss & Improvement Table', fontsize=18, fontweight='bold', pad=30)
    plt.tight_layout()
    plt.savefig(output_path, dpi=300, bbox_inches='tight', facecolor='white')
    plt.close()
    print(f"Table image saved as {output_path}")

if __name__ == "__main__":
    benchmarks_data = print_metrics_for_all_files_and_collect()
    for i in benchmarks_data:
        if ("fir") in i['benchmark']:
            print(i)
        print()
    generate_comparison_table_image(benchmarks_data)
