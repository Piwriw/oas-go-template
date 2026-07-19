#!/bin/sh

# Verify the exact toolchain, explicit Docker image tags, and immutable Action
# references used by local and CI builds.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

go_version=$(awk '$1 == "go" { print $2; exit }' go.mod)
actual_go_version=$(go env GOVERSION)
if [ "$actual_go_version" != "go$go_version" ]; then
	echo "Go toolchain mismatch: go.mod requires go$go_version, running $actual_go_version" >&2
	exit 1
fi

check_image_tag() {
	ref=$1
	if ! printf '%s\n' "$ref" | grep -Eq '^[^:@[:space:]]+:[^:@[:space:]]+$'; then
		echo "Invalid Docker image tag reference: $ref" >&2
		exit 1
	fi
}

for dockerfile in build/Dockerfile web/Dockerfile; do
	syntax_ref=$(sed -n 's/^# syntax=//p' "$dockerfile" | head -n 1)
	check_image_tag "$syntax_ref"
	for image_ref in $(awk '$1 == "FROM" { print $2 }' "$dockerfile"); do
		check_image_tag "$image_ref"
	done
done

expected_builder="golang:${go_version}-alpine"
builder_ref=$(awk '$1 == "FROM" && $2 ~ /^golang:/ { print $2; exit }' build/Dockerfile)
case "$builder_ref" in
	"$expected_builder") ;;
	*)
		echo "Docker Go image does not match go.mod: $builder_ref (want $expected_builder)" >&2
		exit 1
		;;
esac

for action_ref in $(sed -n 's/^[[:space:]]*- uses: \([^[:space:]#]*\).*$/\1/p' .github/workflows/*.yml); do
	case "$action_ref" in
		./*) ;;
		*@????????????????????????????????????????)
			if ! printf '%s\n' "$action_ref" | grep -Eq '@[0-9a-f]{40}$'; then
				echo "Invalid GitHub Action SHA: $action_ref" >&2
				exit 1
			fi
			;;
		*)
			echo "Mutable GitHub Action reference: $action_ref" >&2
			exit 1
			;;
	esac
done

echo "Go toolchain is $actual_go_version; Docker tags and GitHub Action SHAs are valid."
