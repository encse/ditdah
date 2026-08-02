# Pure Go Morse LSTM

This project runs an exported PyTorch model without PyTorch, ONNX Runtime, Gorgonia, CGO, or native libraries.

It implements:

- four Linear + ReLU projection layers;
- one PyTorch-compatible LSTM layer;
- the five-class frame classifier;
- comparison against PyTorch reference tensors.

To export the model from python:
```
python -m morse_timing.export_onnx 0728/models/morse-lstm-curriculum.working-stage-063.pt --output qqq
```

Then replace the contents of `decoder/model` with `qqq`. 


Run tests with 
```
go test ./...
```

## Run

```bash
go run .
```


