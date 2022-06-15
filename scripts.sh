#!/bin/bash

packages=(
	'.'
	'http/framework/echo'
	'http/trace/opencensus'
)
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )";

function test_packages () {
	echo "Testing packages..."
	for i in "${packages[@]}"; do
		pushd $SCRIPT_DIR/$i &> /dev/null
		echo -e "\nTesting: $(go list -m)"
		go test ./...
		popd &> /dev/null
	done
}

function tidy_packages () {
	echo "Tidying packages..."
	for i in "${packages[@]}"; do
		pushd $SCRIPT_DIR/$i &> /dev/null
		echo -e "\nTidying: $(go list -m)"
		go mod tidy
		popd &> /dev/null
	done
}


## Choose which function to use by argument
case $1 in
	t | test)
		test_packages
		;;
	tidy)
		tidy_packages
		;;
	*)
		echo "Usage: $(basename $0) [OPTIONS]

Options:
-t, --test        Runs 'go test ./...' for ctxerr and all subpackages
--tidy            Runs 'go mod tidy ./...' for ctxerr and all subpackages"
		;;
esac