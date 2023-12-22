#!/bin/bash

set -e -o pipefail

cd rule-set
git init
git config --local user.email "github-action@users.noreply.github.com"
git config --local user.name "GitHub Action"
git remote add origin https://github-action:$GITHUB_TOKEN@github.com/aoxiang1221/rule-set.git
git branch -M geosite
git add .
git commit -m "Update geosite rule-set at $(date +%Y-%m-%d)"
git push -f origin geosite