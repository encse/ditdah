#!/bin/sh

set -eu

hamlib_version="4.7.2"
hamlib_sha256="ae1fcf2dbc80ea0786ea8f047b09399c3f7737d1930442f61a031708ed33e88f"
hamlib_url="https://github.com/Hamlib/Hamlib/releases/download/${hamlib_version}/hamlib-${hamlib_version}.tar.gz"
build_recipe="static-without-libusb-v3"

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_dir=$(dirname "$script_dir")
build_root=${DITDAH_HAMLIB_BUILD_ROOT:-"${repository_dir}/.build/hamlib"}
source_dir=${DITDAH_HAMLIB_SOURCE_DIR:-}
custom_source=false

if [ -n "$source_dir" ]; then
    custom_source=true
    source_dir=$(CDPATH= cd -- "$source_dir" && pwd)
    if [ ! -x "${source_dir}/configure" ]; then
        echo "Hamlib source directory does not contain an executable configure script: $source_dir" >&2
        exit 1
    fi
fi

case "$(uname -s):$(uname -m)" in
    Darwin:arm64)
        target="darwin-arm64"
        ;;
    Darwin:x86_64)
        target="darwin-amd64"
        ;;
    Linux:x86_64)
        target="linux-amd64"
        ;;
    MINGW64_NT-*:x86_64|MSYS_NT-*:x86_64)
        target="windows-amd64"
        ;;
    *)
        echo "unsupported platform: $(uname -s) $(uname -m)" >&2
        exit 1
        ;;
esac

for command_name in curl make tar; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
        echo "required command not found: $command_name" >&2
        exit 1
    fi
done

if command -v sha256sum >/dev/null 2>&1; then
    checksum_command="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    checksum_command="shasum -a 256"
else
    echo "required command not found: sha256sum or shasum" >&2
    exit 1
fi

download_dir="${build_root}/downloads"
install_dir="${build_root}/${hamlib_version}/${target}"
archive="${download_dir}/hamlib-${hamlib_version}.tar.gz"
library="${install_dir}/lib/libhamlib.a"
build_stamp="${install_dir}/.ditdah-build"

mkdir -p "$download_dir" "$install_dir"

if [ -z "$source_dir" ] && [ -f "$library" ] && [ -f "$build_stamp" ] &&
    [ "$(sed -n '1p' "$build_stamp")" = "$build_recipe" ]; then
    echo "Hamlib ${hamlib_version} static library:"
    echo "$library"
    exit 0
fi

if [ -z "$source_dir" ]; then
    if [ ! -f "$archive" ]; then
        echo "Downloading Hamlib ${hamlib_version}"
        curl --fail --location --retry 3 --output "${archive}.download" "$hamlib_url"
        mv "${archive}.download" "$archive"
    fi

    actual_sha256=$($checksum_command "$archive" | awk '{print $1}')
    if [ "$actual_sha256" != "$hamlib_sha256" ]; then
        echo "Hamlib archive checksum mismatch" >&2
        echo "expected: $hamlib_sha256" >&2
        echo "actual:   $actual_sha256" >&2
        exit 1
    fi
fi

work_dir=$(mktemp -d "${build_root}/work.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

if [ -z "$source_dir" ]; then
    tar -xzf "$archive" -C "$work_dir"
    source_dir="${work_dir}/hamlib-${hamlib_version}"
fi
mkdir -p "${work_dir}/build"
cd "${work_dir}/build"

"${source_dir}/configure" \
    --prefix="$install_dir" \
    --disable-shared \
    --enable-static \
    --without-cxx-binding \
    --without-libusb

logical_cpus=$(getconf _NPROCESSORS_ONLN 2>/dev/null || sysctl -n hw.logicalcpu 2>/dev/null || echo 1)
make -j "$logical_cpus"
make install

# Hamlib's LICENSE refers to AUTHORS, which `make install` does not install.
cp "${source_dir}/AUTHORS" "${install_dir}/share/doc/hamlib/AUTHORS"

if [ ! -f "$library" ]; then
    echo "Hamlib build did not produce $library" >&2
    exit 1
fi

if [ "$custom_source" = true ]; then
    printf '%s\n' "custom-source" > "$build_stamp"
else
    printf '%s\n' "$build_recipe" > "$build_stamp"
fi

echo "Hamlib ${hamlib_version} static library:"
echo "$library"
