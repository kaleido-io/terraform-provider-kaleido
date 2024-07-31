#!/bin/bash
export GOBIN="$(PWD)/bin";

go install golang.org/x/vuln/cmd/govulncheck@latest

vulncheckOuput=$("$GOBIN"/govulncheck -test -json ./... | jq '.vulnerability.osv.id');
foundVul=false;

# loop through command output
while read -r line; do
  # check if line is in array
    if [[ "$line" != "null" ]]; then
        if grep -q "^${line//\"/}" "./.govulnignore" && ! grep -q "^#" <<< "${line//\"/}";  then
            printf "Skipped vulnerability: $line as it's in the skipped list. \n";
        else
            printf "! Found new vulnerability: ${line}. \n";
            foundVul=true;
        fi
    fi
done <<< "$vulncheckOuput"

if [[ $foundVul == true ]]; then
    printf "!!! New vulnerability found, running govulncheck in plaintext mode to print out the issue.\n#### Go Vulnerability check found new issue ####\n" && "$GOBIN"/govulncheck -test ./...
fi