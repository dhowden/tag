#!/usr/bin/env bash

env GOOS=linux GOARCH=amd64 go build -o /Users/ilko/go/bin/tag-linux_amd64
env GOOS=darwin GOARCH=amd64 go build -o /Users/ilko/go/bin/tag-darwin_amd64
