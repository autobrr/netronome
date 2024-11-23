#!/bin/bash

# ============================
# Configuration
# ============================

# Parse command line arguments
backup=false
while getopts "b" opt; do
    case $opt in
    b)
        backup=true
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    esac
done

# Directories to search
directories=("./internal" "./web" "./cmd")

# File extensions to target
extensions=("go" "ts" "tsx" "css")

# License headers for different file types
read -r -d '' HEADER_GO <<'EOM'
// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

EOM

read -r -d '' HEADER_FRONTEND <<'EOM'
/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

EOM

# Common files/paths to exclude
declare -a EXCLUDES=(
    # Go
    "go.mod"
    "go.sum"
    "vendor/"
    "mock_"

    # Frontend
    "node_modules/"
    "dist/"
    "build/"
    ".next/"
    "package.json"
    "package-lock.json"
    "pnpm-lock.yaml"
    "tailwind.config."
    "vite.config."
    "postcss.config."
    ".d.ts"
    ".env"
    "index.html"
    "public/"
)

# ============================
# Function Definitions
# ============================

# Function to check if file should be excluded
should_exclude() {
    local file="$1"
    for exclude in "${EXCLUDES[@]}"; do
        if [[ "$file" == *"$exclude"* ]]; then
            return 0 # true, should exclude
        fi
    done
    return 1 # false, should not exclude
}

# Function to check if file has any kind of copyright header
has_copyright_header() {
    local file="$1"
    # Check first 5 lines of the file for common copyright indicators
    head -n 5 "$file" | grep -iE "copyright|license|spdx|©" >/dev/null
    return $?
}

# Function to prepend header to a file
add_header() {
    local file="$1"
    local header="$2"
    # Create a temporary file
    tmp_file=$(mktemp) || exit 1
    # Write header and then the original file content
    printf "%s\n\n" "$header" >"$tmp_file"
    cat "$file" >>"$tmp_file"
    # Replace the original file with the new file
    mv "$tmp_file" "$file"
}

# Function to get appropriate header based on file extension
get_header() {
    local ext="$1"
    if [[ "$ext" == "go" ]]; then
        echo "$HEADER_GO"
    else
        echo "$HEADER_FRONTEND"
    fi
}

# Function to check if the exact header already exists
has_exact_header() {
    local file="$1"
    local ext="$2"
    local header

    if [[ "$ext" == "go" ]]; then
        header="$HEADER_GO"
    else
        header="$HEADER_FRONTEND"
    fi

    # Remove trailing newlines from header for comparison
    local header_clean=$(echo "$header" | sed -e 's/[[:space:]]*$//')
    # Check if the header exists at the start of the file
    local file_start=$(head -n 4 "$file" | sed -e 's/[[:space:]]*$//')
    [[ "$file_start" == *"$header_clean"* ]]
    return $?
}

# ============================
# Main Logic
# ============================

files_processed=0
files_skipped=0
files_excluded=0

for dir in "${directories[@]}"; do
    for ext in "${extensions[@]}"; do
        # Find all files with the given extension in the directory
        while IFS= read -r -d '' file; do
            # Skip excluded files
            if should_exclude "$file"; then
                echo "Excluding: $file"
                ((files_excluded++))
                continue
            fi

            # Skip if exact header already exists
            if has_exact_header "$file" "$ext"; then
                echo "Skipping (has exact header): $file"
                ((files_skipped++))
                continue
            fi

            # Skip if any copyright header exists
            if has_copyright_header "$file"; then
                echo "Skipping (has other header): $file"
                ((files_skipped++))
                continue
            fi

            # Get appropriate header for file type
            header=$(get_header "$ext")

            echo "Adding header to: $file"

            # Create a backup if enabled
            if [ "$backup" = "true" ]; then
                cp "$file" "$file.bak"
            fi

            # Add the header
            add_header "$file" "$header"
            ((files_processed++))

        done < <(find "$dir" -type f -name "*.$ext" -print0)
    done
done

echo "======================================="
echo "License Header Addition Complete"
echo "---------------------------------------"
echo "Files processed: $files_processed"
echo "Files skipped (had headers): $files_skipped"
echo "Files excluded: $files_excluded"
echo "======================================="
