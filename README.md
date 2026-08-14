# Pure Go Morse LSTM

This project runs an exported PyTorch model without PyTorch, ONNX Runtime, Gorgonia, CGO, or native libraries.

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

## Logbook database

The logbook uses SQLite through the pure-Go `modernc.org/sqlite` driver. Its
schema is defined in `internal/logbook/schema.sql`, while application queries
use Jet's generated, type-safe SQL builder instead of raw SQL strings.

After changing the schema, regenerate the Jet table definitions:

```bash
go generate ./internal/logbook
```

The generator:

1. creates an in-memory SQLite database;
2. applies `internal/logbook/schema.sql`;
3. inspects the resulting database schema with Jet;
4. regenerates `internal/logbook/dbgen/table` and `dbgen/model`.

The generated Jet model is used only as an internal persistence DTO. The rest
of the application uses its own value-based `QSO` model through the `Store`
interface, so Jet's nullable pointers do not escape the SQLite implementation.
Commit the generated `dbgen` files together with every schema change. The
generator has the `tools` build tag and is not included in the application
binary.

## Run

```bash
go run .
```
