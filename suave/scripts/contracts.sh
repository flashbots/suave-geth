#!/bin/sh

# Resolve the current folder for the script and move
# from there to the suave folder
SCRIPT_DIR="$(dirname "$(realpath "$0")")"
cd "$SCRIPT_DIR"/..

# Function to compile the contracts using 'forge' and converting
# the artifacts into a simplified format that only stores ai, bytecode and deployedBytecode
build() {
    forge build
    find artifacts -type f -name "*.json" -exec sh -c 'jq "{ \"abi\": .abi, \"deployedBytecode\": .deployedBytecode.object, \"bytecode\": .bytecode.object }" "$1" > "$1.tmp" && mv "$1.tmp" "$1"' sh {} \;
}

# Function to clean the artifacts
clean() {
    forge clean
}

# Main function to call other functions based on the provided command
main() {
  case $1 in
    build) build ;;
    clean) clean ;;
    *) echo "Invalid command. Available commands: build, clean" ;;
  esac
}

# Call the main function with the first argument provided
main "$1"
