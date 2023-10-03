#!/bin/bash

# Define the version you want to extract
target_version=$1

if [ -z "$target_version" ]; then
    echo "Target version is empty"
    exit 1
fi

# Input changelog file
changelog_file="CHANGELOG.md"

# Output markdown file
output_file="Release_${target_version}.md"

# Function to add entries to the markdown file
add_entry() {
    echo -e "$1"
}

# Flag to start processing when the target version is found
found_version=false

# Read the changelog file line by line
while IFS= read -r line; do
    # Check if the line starts with the target version
    if [[ $line == "## $target_version"* ]]; then
        echo $line
        found_version=true
        continue
    elif [[ $line == "##"* ]]; then
        # We moved to another version, stop processing
        found_version=false
    fi

    # If we've found the target version, start processing entries
    if [ "$found_version" = true ]; then
        # Add the current entry to the markdown file
        add_entry "$line"
    fi
done < "$changelog_file"