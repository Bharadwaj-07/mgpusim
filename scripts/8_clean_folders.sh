#!/bin/bash

# Function to delete files in a directory
clean_directory() {
    local dir="$1"
    
    # Find and delete files that are not .gitignore or .go
    find "$dir" -type f ! -name '.gitignore' ! -name '*.go' -exec rm -f {} +
}

# Export the function to use it with find
export -f clean_directory

# Traverse all subdirectories except "runner" and "server"
find . -type d ! -name 'runner' ! -name 'server'  -exec bash -c 'clean_directory "$0"' {} \;

echo "Cleanup complete."
