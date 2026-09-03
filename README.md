# DitDah

DitDah is a terminal companion for amateur-radio operators, built to support
contacts from Morse decoding to logging.

[Website](https://encse.github.io/ditdah/) ·
[Downloads](https://github.com/encse/ditdah/releases) ·
[Roadmap](ROADMAP.md)

## Features

- Manage QSOs in a built-in logbook and synchronize them with QRZ Logbook.
- Decode Morse code from live audio input.
- Track the connected radio's frequency and automatically fill it into new
  QSOs.
- Use the full interface with either the keyboard or mouse.

## Installation

### Download a release

Download the archive for your operating system from
[GitHub Releases](https://github.com/encse/ditdah/releases), extract it, and
run `ditdah` from a terminal.

Linux and Windows releases target x86-64. macOS releases are available for both
Apple Silicon and Intel; the Apple Silicon build uses the Go SIMD experiment
when the toolchain supports it and falls back to the scalar implementation
otherwise.

DitDah stores its logbook in the platform's user application-data directory:
`~/Library/Application Support/ditdah` on macOS, `%LOCALAPPDATA%\ditdah` on
Windows, and `$XDG_DATA_HOME/ditdah` (normally `~/.local/share/ditdah`) on
Linux.

### Build from source

Building requires:

- Go 1.27 or newer;
- a C compiler supported by Go's cgo toolchain;
- a working system audio input backend.

```bash
git clone https://github.com/encse/ditdah.git
cd ditdah
./scripts/build-hamlib.sh
go build -o ditdah .
./ditdah
```

On Windows, build and run `ditdah.exe` instead.

Print the embedded build version with `ditdah --version`.

For a quick development run without creating a binary:

```bash
go run .
```

The build requires the pinned static Hamlib dependency. Build it with:

```bash
./scripts/build-hamlib.sh
```

The script downloads the verified Hamlib source archive and installs the
headers and static library under `.build/hamlib`. Direct USB backends are
disabled; radios exposed by the operating system as serial ports remain
supported. The script supports amd64 Linux and Windows (from an MSYS2 MINGW64
shell), plus amd64 and arm64 macOS. cgo links this library into the resulting
DitDah executable.

To rebuild DitDah against a modified Hamlib source tree, pass its path to the
script:

```bash
DITDAH_HAMLIB_SOURCE_DIR=/path/to/hamlib ./scripts/build-hamlib.sh
go build -o ditdah .
```

Hamlib licensing and source details are recorded in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

## First run

1. Start DitDah in a terminal.
2. Enter your station callsign and select the audio input connected to your
   receiver.
3. Optionally configure QRZ.com credentials.
4. Press `F1` for the Morse decoder or `F2` for the logbook.

Your terminal may request microphone permission when the decoder first opens.
The selected input is remembered in `logbook.db`.

## Development

Run the complete test suite:

```bash
go test ./...
```

Run concurrency-sensitive tests with the race detector:

```bash
go test -race ./...
```

Regenerate the application screenshots used by the website:

```bash
vhs screenshots.tape
```

This requires [VHS](https://github.com/charmbracelet/vhs).

Database changes are versioned in `internal/database/migrations`. After adding a
migration, regenerate the Jet database model and run the tests:

```bash
go generate ./internal/database
go test ./...
```

Create a release from **Actions → Release → Run workflow**. Enter a version
without the `v` prefix, such as `1.2.0`. The workflow validates the version,
builds and tests every supported platform, then creates the `v1.2.0` tag and
publishes the GitHub Release from the selected commit.

Technical decisions and planned work are tracked in [ROADMAP.md](ROADMAP.md).
