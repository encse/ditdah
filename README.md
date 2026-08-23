# DitDah

DitDah is a local-first terminal application for amateur-radio operators.
It combines a searchable QSO logbook with live Morse decoding, callsign lookup,
and optional QRZ.com synchronization in one keyboard- and mouse-friendly
interface.

[Website](https://encse.github.io/ditdah/) ·
[Downloads](https://github.com/encse/ditdah/releases) ·
[Roadmap](ROADMAP.md)

## Features

- Create, edit, search, and delete QSOs.
- Record callsigns, date and time, frequency, mode, reports, exchanges, name,
  QTH, and notes.
- Decode Morse code from a selectable live audio input.
- Maintain a callsign list beside the decoder and highlight the selected
  callsign in decoded text.
- Look up callsign details through QRZ.com and reuse cached results.
- Upload pending QSOs to QRZ Logbook and replace previously synchronized records
  after edits.
- Keep the logbook and settings in a local SQLite database.
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

DitDah creates `logbook.db` in the directory from which it is started. Run
it from a stable directory if you want the application to keep using the same
logbook.

### Build from source

Building requires:

- Go 1.27 or newer;
- a C compiler supported by Go's cgo toolchain;
- a working system audio input backend.

```bash
git clone https://github.com/encse/ditdah.git
cd ditdah
go build -o ditdah .
./ditdah
```

On Windows, build and run `ditdah.exe` instead.

Print the embedded build version with `ditdah --version`.

For a quick development run without creating a binary:

```bash
go run .
```

## First run

1. Start DitDah in a terminal.
2. Open the `[=]` menu and select **Settings**.
3. Enter your station callsign and select the audio input connected to your
   receiver.
4. Optionally configure QRZ.com credentials.
5. Press `F1` for the logbook or `F2` for the Morse decoder.

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
go test -count=1 -tags screenshots -run '^TestScreenshot' \
  ./internal/logbook/tui ./internal/qsoeditor/tui ./internal/decoder/tui
```

Database changes are versioned in `internal/database/migrations`. After adding a
migration, regenerate the Jet database model and run the tests:

```bash
go generate ./internal/database
go test ./...
```

Technical decisions and planned work are tracked in [ROADMAP.md](ROADMAP.md).
