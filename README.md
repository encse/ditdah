# Morse Manual

Morse Manual is a terminal application for amateur-radio operation. The
current interface provides a searchable SQLite logbook with a QSO table and a
detail view. A live Morse decoder is also under development.

## Run

```bash
go run .
```

The logbook is stored in `logbook.db` in the current directory.

Inside the logbook, use the arrow keys or `j`/`k` to move, `/` to search, and
`q` to quit.

## Decoder development

The decoder runs an exported PyTorch model without PyTorch, ONNX Runtime,
Gorgonia, CGO, or native inference libraries.

It implements:

- four Linear + ReLU projection layers;
- one PyTorch-compatible LSTM layer;
- the five-class frame classifier;
- comparison against PyTorch reference tensors.

To export the model from python:

```
python -m morse_timing.export_onnx models/final.pt --output qqq
```

Then replace the contents of `decoder/model` with `qqq`. 


Run tests with 
```
go test ./...
```

or
```
GOEXPERIMENT=simd go test ./... -v
```

## Schema changes

Add database changes as a new, versioned Goose migration in
`internal/database/migrations`. Then regenerate the Jet database model:

```bash
go generate ./internal/database
```

Commit the migration and the regenerated `internal/database/dbgen` files
together, and run the tests before committing:

```bash
go test ./...
```
