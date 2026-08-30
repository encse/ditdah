#!/bin/sh

set -eu

hamlib_version="4.7.2"
hamlib_sha256="ae1fcf2dbc80ea0786ea8f047b09399c3f7737d1930442f61a031708ed33e88f"
hamlib_url="https://github.com/Hamlib/Hamlib/releases/download/${hamlib_version}/hamlib-${hamlib_version}.tar.gz"
build_recipe="static-without-libusb-v1"

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_dir=$(dirname "$script_dir")
build_root=${DITDAH_HAMLIB_BUILD_ROOT:-"${repository_dir}/.build/hamlib"}

if [ "$(uname -s)" != "Darwin" ]; then
    echo "build-hamlib.sh currently supports macOS only" >&2
    exit 1
fi

case "$(uname -m)" in
    arm64)
        target="darwin-arm64"
        ;;
    x86_64)
        target="darwin-amd64"
        ;;
    *)
        echo "unsupported macOS architecture: $(uname -m)" >&2
        exit 1
        ;;
esac

for command_name in curl make shasum tar; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
        echo "required command not found: $command_name" >&2
        exit 1
    fi
done

download_dir="${build_root}/downloads"
install_dir="${build_root}/${hamlib_version}/${target}"
archive="${download_dir}/hamlib-${hamlib_version}.tar.gz"
library="${install_dir}/lib/libhamlib.a"
build_stamp="${install_dir}/.ditdah-build"

mkdir -p "$download_dir" "$install_dir"

if [ -f "$library" ] && [ -f "$build_stamp" ] &&
    [ "$(sed -n '1p' "$build_stamp")" = "$build_recipe" ]; then
    echo "Hamlib ${hamlib_version} static library:"
    echo "$library"
    exit 0
fi

if [ ! -f "$archive" ]; then
    echo "Downloading Hamlib ${hamlib_version}"
    curl --fail --location --retry 3 --output "${archive}.download" "$hamlib_url"
    mv "${archive}.download" "$archive"
fi

actual_sha256=$(shasum -a 256 "$archive" | awk '{print $1}')
if [ "$actual_sha256" != "$hamlib_sha256" ]; then
    echo "Hamlib archive checksum mismatch" >&2
    echo "expected: $hamlib_sha256" >&2
    echo "actual:   $actual_sha256" >&2
    exit 1
fi

work_dir=$(mktemp -d "${build_root}/work.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

tar -xzf "$archive" -C "$work_dir"
mkdir -p "${work_dir}/build"
cd "${work_dir}/build"

"${work_dir}/hamlib-${hamlib_version}/configure" \
    --prefix="$install_dir" \
    --disable-shared \
    --enable-static \
    --without-cxx-binding \
    --without-libusb

logical_cpus=$(sysctl -n hw.logicalcpu 2>/dev/null || echo 1)
make -j "$logical_cpus"
make install

if [ ! -f "$library" ]; then
    echo "Hamlib build did not produce $library" >&2
    exit 1
fi

printf '%s\n' "$build_recipe" > "$build_stamp"

echo "Hamlib ${hamlib_version} static library:"
echo "$library"
