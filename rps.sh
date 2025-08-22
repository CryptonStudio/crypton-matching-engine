#!/bin/bash
go build -o _rps ./cmd/rps
./_rps "$@"