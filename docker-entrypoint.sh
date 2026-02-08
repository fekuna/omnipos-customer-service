#!/bin/sh

# If arguments are passed, run them
if [ "$#" -gt 0 ]; then
    exec "$@"
fi

# Default to running air
exec air
