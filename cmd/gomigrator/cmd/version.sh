#!/bin/sh
date -u '+%Y-%m-%dT%H:%M:%S' > _build_date.txt
git log --format="%h" -n 1 > _githash.txt
