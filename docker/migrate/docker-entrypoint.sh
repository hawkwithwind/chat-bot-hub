#!/bin/sh
set -e

exec su-exec migrateuser /migrate "$@"
