#!/bin/bash

# Configuration arrays
configs=("l2-prefetcher")
benchmarks=(
  "pagerank"
  "bitonicsort"
  "fir"
  # "floydwarshall"
  # "matrixmultiplication"
  # "matrixtranspose"
  # "simpleconvolution"
  # "kmeans"
  # "spmv"
  # "stencil2d"
)

# Array to track which benchmarks have been started
declare -A benchmark_status

# Initialize benchmark status
for benchmark in "${benchmarks[@]}"; do
  benchmark_status["$benchmark"]=0 # 0 = not started, 1 = running or completed
done

# Function to check available RAM in GB
check_ram() {
  free -g | awk '/^Mem:/{print $7}'
}

# Function to run a benchmark with logging
run_benchmark() {
  local config="$1"
  local benchmark="$2"

  # Mark this benchmark as started
  benchmark_status["$benchmark"]=1

  (
    cd "$config" || exit 1
    echo "Starting benchmark: $benchmark at $(TZ='India/Kokata' date)" >&2
    bash "${benchmark}.sh" >"${benchmark}.log" 2>&1
    echo "Finished benchmark: $benchmark at $(TZ='India/Kolkata' date)" >&2
  ) &
}

# Function to count running benchmarks
count_running_benchmarks() {
  jobs -r | wc -l
}

# Function to get next unstarted benchmark
get_next_benchmark() {
  for benchmark in "${benchmarks[@]}"; do
    if [ "${benchmark_status[$benchmark]}" -eq 0 ]; then
      echo "$benchmark"
      return 0
    fi
  done
  echo "" # Return empty string if no benchmarks are left
}

# Main loop to check RAM and schedule benchmarks
scheduled_benchmarks=0
total_benchmarks=${#benchmarks[@]}

echo "Starting benchmark scheduler. Total benchmarks to run: $total_benchmarks"

while [ $scheduled_benchmarks -lt $total_benchmarks ]; do
  # Get available RAM and calculate how many benchmarks we can run
  available_ram=$(check_ram)
  echo "Available RAM: ${available_ram}GB"

  # Calculate number of benchmarks to run based on RAM (10GB per benchmark)
  num_benchmarks=$((available_ram / 4))

  # Cap at maximum 3 benchmarks, minimum 1
  if [ $num_benchmarks -gt 2 ]; then
    num_benchmarks=2
  elif [ $num_benchmarks -lt 1 ]; then
    num_benchmarks=1
  fi

  echo "Target number of concurrent benchmarks: $num_benchmarks"

  # Check current running benchmarks
  running_benchmarks=$(count_running_benchmarks)
  echo "Currently running benchmarks: $running_benchmarks"

  # Schedule new benchmarks if there's capacity
  for config in "${configs[@]}"; do
    while [ $running_benchmarks -lt $num_benchmarks ] && [ $scheduled_benchmarks -lt $total_benchmarks ]; do
      # Get next unstarted benchmark
      next_benchmark=$(get_next_benchmark)

      # If no more benchmarks to run, break
      if [ -z "$next_benchmark" ]; then
        break
      fi

      echo "Scheduling benchmark: $next_benchmark"
      run_benchmark "$config" "$next_benchmark"

      scheduled_benchmarks=$((scheduled_benchmarks + 1))
      running_benchmarks=$((running_benchmarks + 1))

      echo "Progress: $scheduled_benchmarks/$total_benchmarks benchmarks scheduled"
    done
  done

  # If all benchmarks have been scheduled, break the loop
  if [ $scheduled_benchmarks -ge $total_benchmarks ]; then
    break
  fi

  # Wait before checking again
  echo "Waiting 10 seconds before next check..."
  sleep 10
done

# Wait for all remaining benchmarks to complete
echo "All benchmarks scheduled. Waiting for completion..."
wait
echo "All benchmarks completed at $(date)"

# Print summary of completed benchmarks
echo "Benchmark execution summary:"
for benchmark in "${benchmarks[@]}"; do
  if [ -f "${configs[0]}/${benchmark}.log" ]; then
    echo "✓ $benchmark completed"
  else
    echo "✗ $benchmark failed or did not run"
  fi
done
