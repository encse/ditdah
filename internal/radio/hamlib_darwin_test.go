//go:build darwin && cgo

package radio

import "testing"

func TestHamlibListsSerialRadios(t *testing.T) {
	models, err := New().Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}
	if len(models) == 0 {
		t.Fatal("Models() returned no serial radios")
	}
	for _, model := range models {
		if model.ID <= 0 || model.Name == "" || model.DefaultBaudRate <= 0 {
			t.Fatalf("invalid Hamlib model = %#v", model)
		}
	}
}
