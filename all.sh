#!/usr/bin/env bash

# Tests Go projects.  There must be project cmd sub directories where this script is executed.
# Assumes a flat directory hierarchy below cmd (the maxdepth in the find command).
# Runs go test for each cmd sub directory.  This will check the code compiles even when there are
# no test files.
# If there is an env.list file in the sub project then this will be used to set env var before running
# go test and unset them after.  This avoids accidental cross project dependencies from env.list files.
#
# usage: ./all.sh

set -e

if [ ! -f all.sh ]; then
	echo 'all.sh must be run from the project root' 1>&2
	exit 1
fi

projects=`find cmd -maxdepth 2 -name '*.go' -print | awk -F "/" '{print $1 "/" $2}' | sort -u | egrep -v vendor`

for i in ${projects[@]}; do
	if [ -f ${i}/env.list ]; then
		export $(cat ${i}/env.list | grep = | xargs)
	fi

	go test  -v ./${i}

	if [ -f ${i}/env.list ]; then
		unset $(cat ${i}/env.list | grep = | awk -F "=" '{print $1}')
	fi
done
