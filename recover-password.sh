#!/usr/bin/env sh
set -e
cd "$(dirname "$0")"
echo
echo "============================================"
echo " ResumeTrail password recovery"
echo "============================================"
echo
go run ./cmd/recover-password
