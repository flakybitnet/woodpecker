#!/bin/sh
set -eu

retry () {
    local retries=$1
    shift

    local count=0
    local wait=5
    until "$@"; do
        exit=$?
        wait=$(($wait * 2))
        count=$(($count + 1))
        if [ $count -lt $retries ]; then
            echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
            sleep $wait
        else
            echo "Retry $count/$retries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}