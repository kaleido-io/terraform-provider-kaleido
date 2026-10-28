#!/bin/sh

echo "Ensure trivy is installed..."

if [ ! -x "$(command -v trivy)" ]; then
	case "$OSTYPE" in
		darwin*)
			brew install trivy
			;;
		*)
			curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin latest
			;;
	esac
fi

TMP_DIR=$(mktemp -d)
# copy all the directories specified in the CLI args $1 $2 ... to the tmp dir
for target in "$@"
do
	echo "Copying $target to $TMP_DIR"
	cp -r $target $TMP_DIR
done

echo "Generating SBOM..."
trivy fs --format spdx-json --output sbom.spdx.json $TMP_DIR
rm -rf $TMP_DIR

set -e
echo "Scanning SBOM for Vulnerabilities..."
trivy sbom sbom.spdx.json \
  --db-repository mirror.gcr.io/aquasec/trivy-db:2,ghcr.io/aquasecurity/trivy-db:2,public.ecr.aws/aquasecurity/trivy-db:2 \
  --java-db-repository mirror.gcr.io/aquasec/trivy-java-db:1,ghcr.io/aquasecurity/trivy-java-db:1,public.ecr.aws/aquasecurity/trivy-java-db:1 \
  --severity UNKNOWN,HIGH,CRITICAL --exit-code 1
