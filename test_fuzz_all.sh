#!/usr/bin/env bash

# Inspired by: https://github.com/golang/go/issues/46312#issuecomment-1727928218

set -e

# Fuzz time needs to include the units. For example, 60s is 60 seconds.
fuzzTime=${1:-60s}

files=$(grep -r --include='**_test.go' --files-with-matches 'func Fuzz' .)

if [ -z "${files}" ]; then
    fileCount=0;
else
    fileCount=$(echo "${files}" | wc -l | tr -d '[:space:]');
fi

# Generate some metrics to make output review and estimation of remaining time easier.
testCount=0
for file in ${files}; do
    funcs=$(grep '^func Fuzz' "$file" | sed s/func\ // | sed 's/(.*$//')
    for func in ${funcs}; do
        (( testCount += 1 ))
    done
done

echo -e "\n\n\n###########################################################################"
echo "Running native hardware platform fuzzing"
echo "Found a total of ${testCount} fuzz tests spread across ${fileCount} files"
echo -e "###########################################################################\n"
unset funcs
unset func

echo "Unpacking the fuzz cache into ./testdata/gofuzzcache..."
go run ./cmd/fuzzcache/main.go unpack ./testdata/gofuzzcache ./testdata/fuzzcache

counter=0
for file in ${files}; do
        funcs=$(grep '^func Fuzz' "$file" | sed s/func\ // | sed 's/(.*$//')

        for func in ${funcs}; do
                 (( counter += 1 ))
                echo -e "\nFuzzing ${func} in ${file} (${counter} of ${testCount})"
                parentDir=$(dirname "$file")
                go test -fuzz="$func" -v -tags all -fuzztime="${fuzzTime}" -test.fuzzcachedir "./testdata/gofuzzcache" "$parentDir"
        done
done

# Check for Linux on x86_64 hardware. If found, we also fuzz in 32-bit mode.
if [ "$(uname -s)" == "Linux" ]; then
    if [ "$(uname -m)" == "x86_64" ] || [ "$(uname -m)" == "i686" ]; then
        counter=0

        echo -e "\n\n\n###########################################################################"
        echo "Running 32-bit fuzzing"
        echo -e "###########################################################################\n"
        for file in ${files}; do
            funcs=$(grep '^func Fuzz' "$file" | sed s/func\ // | sed 's/(.*$//')

            for func in ${funcs}; do
                     (( counter += 1 ))
                    echo -e "\n32-bit fuzzing ${func} in ${file} (${counter} of ${testCount})"
                    parentDir=$(dirname "$file")
                    GOARCH=386 go test -v -tags all -fuzz="$func" -fuzztime="${fuzzTime}" "$parentDir"
            done
        done

    fi
fi

echo "Saving the fuzz cache from ./testdata/gofuzzcache..."
go run ./cmd/fuzzcache/main.go pack ./testdata/gofuzzcache ./testdata/fuzzcache
