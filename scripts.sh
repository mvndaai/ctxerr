#!/bin/bash

packages=($(find . -name "go.mod" -print0 | xargs -0 -n1 dirname | sort --unique))
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )";


function cmd_on_packages() {
	echo "$1 packages..."
	for package in "${packages[@]}"; do
		pushd $SCRIPT_DIR/$package &> /dev/null
		echo -e "\n${1}: $(go list -m)"
		for cmd in "${@:2}"; do
			($cmd)
		done
		popd &> /dev/null
	done
}


## Choose which function to use by argument
case $1 in
	t | test)
		cmd_on_packages "Testing" \
			"go clean -testcache" \
			"go test ./..."
		;;
	d | tidy)
		cmd_on_packages "Tidying" "go mod tidy"
		;;
	u | update)
		cmd_on_packages "Updating" \
			"go get github.com/mvndaai/ctxerr" \
			"go get -u ./..." \
			"go mod tidy"
		;;
	*)
		echo "Usage: $(basename $0) [OPTIONS]

Options:
-t, --test          Runs 'go test ./...' for ctxerr and all subpackages
-d, --tidy          Runs 'go mod tidy ./...' for ctxerr and all subpackages
-u, --update        Updates and tidyies ctxerr and all subpackages"
		;;
esac