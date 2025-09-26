#!/bin/bash

# Configuration arrays
configs=("normal" "combined")
benchmarks=(
    "pagerank"
    "bitonicsort"
    "fir"
    "floydwarshall"
    # "kmeans"
    # "matrixmultiplication"
    # "matrixtranspose"
    # "simpleconvolution"
    # "spmv"
    # "stencil2d"
)

# Associative arrays to track benchmark status for each config
declare -A normal_benchmark_status
declare -A prefetcher_benchmark_status

# Initialize benchmark status for both configs
for benchmark in "${benchmarks[@]}"; do
    normal_benchmark_status["$benchmark"]=0
    prefetcher_benchmark_status["$benchmark"]=0
done

# Function to check available RAM in GB
check_ram() {
    free -g | awk '/^Mem:/{print $7}'
}

# Function to run a benchmark with logging
run_benchmark() {
    local config="$1"
    local benchmark="$2"

    # Mark this benchmark as started in appropriate status array
    if [ "$config" == "normal" ]; then
        normal_benchmark_status["$benchmark"]=1
    else
        prefetcher_benchmark_status["$benchmark"]=1
    fi

    (
        cd "$config" || exit 1
        echo "Starting benchmark: $config/$benchmark at $(date)" >&2
        bash "${benchmark}.sh" > "${benchmark}.log" 2>&1
        echo "Finished benchmark: $config/$benchmark at $(date)" >&2
    ) &
}

# Function to count running benchmarks
count_running_benchmarks() {
    jobs -r | wc -l
}

# Function to get next unstarted benchmark for a config
get_next_benchmark() {
    local config="$1"
    local -n status_array="$2"

    for benchmark in "${benchmarks[@]}"; do
        if [ "${status_array[$benchmark]}" -eq 0 ]; then
            echo "$benchmark"
            return 0
        fi
    done
    echo ""
}

# Main execution
total_benchmarks=$((${#benchmarks[@]} * ${#configs[@]}))  # Total benchmarks across all configs
scheduled_benchmarks=0

echo "Starting combined benchmark scheduler. Total benchmarks to run: $total_benchmarks"
echo "RAM allocation: Maximum 6GB per benchmark, Total system RAM: 28GB"

while [ $scheduled_benchmarks -lt $total_benchmarks ]; do
    available_ram=$(check_ram)
    echo "Available RAM: ${available_ram}GB"

    # Calculate number of benchmarks to run (6GB per benchmark)
    num_benchmarks=$((available_ram / 6))

    # Cap at maximum 4 benchmarks (24GB total), minimum 1
    if [ $num_benchmarks -gt 2 ]; then
        num_benchmarks=2
    elif [ $num_benchmarks -lt 1 ]; then
        num_benchmarks=1
    fi

    echo "Target number of concurrent benchmarks: $num_benchmarks"

    # Check current running benchmarks
    running_benchmarks=$(count_running_benchmarks)
    echo "Currently running benchmarks: $running_benchmarks"

    # Try to schedule benchmarks alternating between configs
    while [ $running_benchmarks -lt $num_benchmarks ] && [ $scheduled_benchmarks -lt $total_benchmarks ]; do
        scheduled_this_round=0

        # Try normal config
        if [ $scheduled_this_round -eq 0 ]; then
            next_benchmark=$(get_next_benchmark "normal" normal_benchmark_status)
            if [ -n "$next_benchmark" ]; then
                echo "Scheduling normal benchmark: $next_benchmark"
                run_benchmark "normal" "$next_benchmark"
                scheduled_benchmarks=$((scheduled_benchmarks + 1))
                running_benchmarks=$((running_benchmarks + 1))
                scheduled_this_round=1
                echo "Progress: $scheduled_benchmarks/$total_benchmarks total benchmarks scheduled"
            fi
        fi

        # Try prefetcher config if we still have capacity
        if [ $running_benchmarks -lt $num_benchmarks ] && [ $scheduled_this_round -eq 1 ]; then
            next_benchmark=$(get_next_benchmark "prefetcher" prefetcher_benchmark_status)
            if [ -n "$next_benchmark" ]; then
                echo "Scheduling prefetcher benchmark: $next_benchmark"
                run_benchmark "prefetcher" "$next_benchmark"
                scheduled_benchmarks=$((scheduled_benchmarks + 1))
                running_benchmarks=$((running_benchmarks + 1))
                echo "Progress: $scheduled_benchmarks/$total_benchmarks total benchmarks scheduled"
            fi
        fi

        # If we couldn't schedule any new benchmarks, break
        if [ $scheduled_this_round -eq 0 ]; then
            break
        fi
    done

    # If all benchmarks have been scheduled, break the loop
    if [ $scheduled_benchmarks -ge $total_benchmarks ]; then
        break
    fi

    echo "Waiting 10 seconds before next check..."
    sleep 10
done

# Wait for all remaining benchmarks to complete
echo "All benchmarks scheduled. Waiting for completion..."
wait
echo "All benchmarks completed at $(date)"

# Print summary of completed benchmarks
echo "Benchmark execution summary:"
echo "Normal configuration:"
for benchmark in "${benchmarks[@]}"; do
    if [ -f "normal/${benchmark}.log" ]; then
        echo "✓ normal/$benchmark completed"
    else
        echo "✗ normal/$benchmark failed or did not run"
    fi
done

echo -e "\nPrefetcher configuration:"
for benchmark in "${benchmarks[@]}"; do
    if [ -f "prefetcher/${benchmark}.log" ]; then
        echo "✓ prefetcher/$benchmark completed"
    else
        echo "✗ prefetcher/$benchmark failed or did not run"
    fi
done
