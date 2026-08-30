package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"ditdah/internal/radio"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/rivo/tview"
)

type radioDialog struct {
	modal.Layout
	host       ui.PageHost
	radio      radio.Service
	initial    radio.Config
	models     []radio.Model
	ports      []string
	model      components.SelectField
	port       components.SelectField
	baudRate   components.InputField
	status     components.TextView
	ok         components.Button
	cancel     components.Button
	focusables []tview.Primitive
	handle     modal.Handle
	acceptWith func(radio.Config, uint64)
	generation atomic.Uint64
	checking   bool
}

func newRadioDialog(
	host ui.PageHost,
	controls components.Factory,
	service radio.Service,
	initial radio.Config,
	acceptWith func(radio.Config, uint64),
) *radioDialog {
	controls = controls.Modal()
	dialog := &radioDialog{
		host:       host,
		radio:      service,
		initial:    initial,
		acceptWith: acceptWith,
	}
	dialog.model = controls.SelectField(
		"Radio", nil, -1, settingsLabelWidth, 0,
	)
	dialog.port = controls.SelectField(
		"Serial port", nil, -1, settingsLabelWidth, 0,
	)
	dialog.baudRate = controls.InputField("Baud rate", "")
	dialog.baudRate.SetLabelWidth(settingsLabelWidth)
	dialog.status = controls.TextView()
	dialog.status.SetTextAlign(tview.AlignCenter)
	dialog.ok = controls.Button("OK")
	dialog.cancel = controls.Button("Cancel")
	dialog.model.SetChangedFunc(dialog.modelChanged)
	dialog.port.SetChangedFunc(func(int, string) { dialog.invalidate() })
	dialog.baudRate.SetChangedFunc(func(string) { dialog.invalidate() })
	dialog.ok.SetSelectedFunc(dialog.accept)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{
		dialog.model,
		dialog.port,
		dialog.baudRate,
		dialog.ok,
		dialog.cancel,
	}
	fields := controls.Grid().
		SetRows(1, 1, 1, 1, 1).
		SetColumns(0).
		AddItem(dialog.model, 0, 0, 1, 1, 0, 0, false).
		AddItem(dialog.port, 2, 0, 1, 1, 0, 0, false).
		AddItem(dialog.baudRate, 4, 0, 1, 1, 0, 0, false)
	dialog.Layout = modal.NewLayout(controls, " Radio ", 72).
		Row(fields, 5).
		Spacer().
		Row(dialog.status, 1).
		Actions(centeredButtons(controls, dialog.ok, dialog.cancel))
	return dialog
}

func (d *radioDialog) Focusables() []tview.Primitive { return d.focusables }

func (d *radioDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *radioDialog) Run(ctx context.Context) {
	if d.radio == nil {
		d.host.Update(d, func() {
			d.showError(errors.New("Hamlib is unavailable"))
		})
		<-ctx.Done()
		return
	}
	models, modelErr := d.radio.Models()
	ports, portErr := d.radio.Ports()
	if ctx.Err() != nil {
		return
	}
	d.host.Update(d, func() {
		d.models = models
		d.ports = includeSavedPort(ports, d.initial.Port)
		d.model.SetOptions(modelOptions(models), selectedModel(d.initial.ModelID, models))
		d.port.SetOptions(d.ports, selectedPort(d.initial.Port, d.ports))
		if d.initial.BaudRate > 0 {
			d.baudRate.SetValue(strconv.Itoa(d.initial.BaudRate))
		} else if index, _ := d.model.CurrentOption(); index >= 0 {
			d.baudRate.SetValue(strconv.Itoa(models[index].DefaultBaudRate))
		}
		if err := errors.Join(modelErr, portErr); err != nil {
			d.showError(err)
		} else if len(models) == 0 {
			d.showError(errors.New("no supported serial radio models found"))
		} else if len(d.ports) == 0 {
			d.showError(errors.New("no serial ports found"))
		}
	})
	<-ctx.Done()
}

func (d *radioDialog) modelChanged(index int, _ string) {
	if index >= 0 && index < len(d.models) {
		d.baudRate.SetValue(strconv.Itoa(d.models[index].DefaultBaudRate))
	}
	d.invalidate()
}

func (d *radioDialog) invalidate() {
	d.generation.Add(1)
	if d.status.Text() != "" {
		d.status.SetStyle(components.TextViewMuted)
		d.status.SetText("")
	}
}

func (d *radioDialog) currentConfig() (radio.Config, error) {
	modelIndex, _ := d.model.CurrentOption()
	if modelIndex < 0 || modelIndex >= len(d.models) {
		return radio.Config{}, errors.New("radio model is required")
	}
	portIndex, port := d.port.CurrentOption()
	if portIndex < 0 || strings.TrimSpace(port) == "" {
		return radio.Config{}, errors.New("serial port is required")
	}
	baudRate, err := strconv.Atoi(strings.TrimSpace(d.baudRate.Value()))
	if err != nil || baudRate <= 0 {
		return radio.Config{}, errors.New("baud rate must be a positive number")
	}
	model := d.models[modelIndex]
	return radio.Config{
		ModelID:   model.ID,
		ModelName: modelLabel(model),
		Port:      port,
		BaudRate:  baudRate,
	}, nil
}

func (d *radioDialog) accept() {
	if d.checking {
		return
	}
	config, err := d.currentConfig()
	if err != nil {
		d.showError(err)
		return
	}
	generation := d.generation.Add(1)
	d.checking = true
	d.status.SetStyle(components.TextViewMuted)
	d.status.SetText("Checking...")
	started := d.host.Background(d, func(ctx context.Context) {
		frequency, checkErr := d.radio.Check(ctx, config)
		if ctx.Err() != nil {
			return
		}
		d.host.Update(d, func() {
			d.checking = false
			if generation != d.generation.Load() {
				return
			}
			if checkErr != nil {
				d.showError(checkErr)
				return
			}
			if d.acceptWith != nil {
				d.acceptWith(config, frequency)
			}
			d.close()
		})
	})
	if !started {
		d.checking = false
		d.showError(errors.New("could not start radio connection check"))
	}
}

func (d *radioDialog) showError(err error) {
	d.status.SetStyle(components.TextViewDanger)
	d.status.SetText("Error: " + err.Error())
}

func (d *radioDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func modelOptions(models []radio.Model) []string {
	options := make([]string, len(models))
	for index, model := range models {
		options[index] = modelLabel(model)
	}
	return options
}

func modelLabel(model radio.Model) string {
	return strings.TrimSpace(model.Manufacturer + " " + model.Name)
}

func selectedModel(id int, models []radio.Model) int {
	for index, model := range models {
		if model.ID == id {
			return index
		}
	}
	if id == 0 && len(models) > 0 {
		return 0
	}
	return -1
}

func includeSavedPort(ports []string, saved string) []string {
	for _, port := range ports {
		if port == saved {
			return ports
		}
	}
	if saved == "" {
		return ports
	}
	return append(append([]string(nil), ports...), saved)
}

func selectedPort(saved string, ports []string) int {
	for index, port := range ports {
		if port == saved {
			return index
		}
	}
	if saved == "" && len(ports) == 1 {
		return 0
	}
	return -1
}

func formatFrequency(frequency uint64) string {
	megahertz := frequency / 1_000_000
	fraction := fmt.Sprintf("%06d", frequency%1_000_000)
	for len(fraction) > 3 && fraction[len(fraction)-1] == '0' {
		fraction = fraction[:len(fraction)-1]
	}
	return fmt.Sprintf("%d.%s MHz", megahertz, fraction)
}
