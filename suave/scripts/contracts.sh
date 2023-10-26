#!/bin/sh

# Resolve the current folder for the script and move
# from there to the suave folder
SCRIPT_DIR="$(dirname "$(realpath "$0")")"
cd "$SCRIPT_DIR"/..

# Function to compile the contracts using 'forge' and converting
# the artifacts into a simplified format that only stores ai, bytecode and deployedBytecode
build() {
    forge build
    find artifacts -type f -name "*.json" ! -name "SuaveLib.json" -exec sh -c 'jq "{ \"abi\": .abi, \"deployedBytecode\":{\"object\": .deployedBytecode.object}, \"bytecode\":{\"object\": .bytecode.object} }" "$1" > "$1.tmp" && mv "$1.tmp" "$1"' sh {} \;
}

# Function to clean the artifacts
clean() {
    forge clean
}

# Function to validate that the contract artifacts are valid
civalidate() {
    build

    # Build again and check if there are any changes in the artifacts folder
    if [ "$(git status --porcelain .)" ]; then
      echo "Artifacts have not been generated."
      exit 1
    else
      # No changes
      echo "Artifacts are correct."
    fi
}

# Main function to call other functions based on the provided command
main() {
  case $1 in
    build) build ;;
    clean) clean ;;
    civalidate) civalidate ;;
    *) echo "Invalid command. Available commands: build, clean, civalidate" ;;
  esac
}

# Call the main function with the first argument provided
main "$1"
