#!/bin/bash

# mkdir private;
# mkdir shared;
# mkdir mgvm;
# mkdir mgvm-nobalance/;

# cp -r ../mgpusim/samples private/
# cp -r ../mgpusim/samples shared/
# cp -r ../mgpusim/samples mgvm/
# cp -r ../mgpusim/samples mgvm-nobalance/

# Create the target directories
mkdir -p normal
mkdir -p l1-prefetcher
mkdir -p l2-prefetcher
mkdir -p combined
mkdir -p infinite-l2

# Define the folders to copy
benchmarks=(
    "aes"
    "atax"
    "bfs"
    "bicg"
    "bitonicsort"
    "conv2d"
    "fastwalshtransform"
    "fir"
    "floydwarshall"
    "kmeans"
    "matrixmultiplication"
    "matrixtranspose"
    "nbody"
    "pagerank"
    "relu"
    "simpleconvolution"
    "spmv"
    "stencil2d"
)

# Source directory
source_dir="../samples"

# Copy the specified folders to the 'normal' directory
for benchmark in "${benchmarks[@]}"; do
    if [ -d "$source_dir/$benchmark" ]; then
        cp -r "$source_dir/$benchmark" normal/
    else
        echo "Warning: $source_dir/$benchmark does not exist."
    fi
done

# Copy the specified folders to the 'prefetcher' directory
for benchmark in "${benchmarks[@]}"; do
    if [ -d "$source_dir/$benchmark" ]; then
        cp -r "$source_dir/$benchmark" l1-prefetcher/
    else
        echo "Warning: $source_dir/$benchmark does not exist."
    fi
done

for benchmark in "${benchmarks[@]}"; do
    if [ -d "$source_dir/$benchmark" ]; then
        cp -r "$source_dir/$benchmark" l2-prefetcher/
    else
        echo "Warning: $source_dir/$benchmark does not exist."
    fi
done

for benchmark in "${benchmarks[@]}"; do
    if [ -d "$source_dir/$benchmark" ]; then
        cp -r "$source_dir/$benchmark" combined/
    else
        echo "Warning: $source_dir/$benchmark does not exist."
    fi
done

for benchmark in "${benchmarks[@]}"; do
    if [ -d "$source_dir/$benchmark" ]; then
        cp -r "$source_dir/$benchmark" infinite-l2/
    else
        echo "Warning: $source_dir/$benchmark does not exist."
    fi
done

echo "Copying complete."
