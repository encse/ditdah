//go:build (darwin || linux || windows) && cgo

package radio

/*
#cgo darwin,arm64 CFLAGS: -I${SRCDIR}/../../.build/hamlib/4.7.2/darwin-arm64/include
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/../../.build/hamlib/4.7.2/darwin-arm64/lib/libhamlib.a
#cgo darwin,amd64 CFLAGS: -I${SRCDIR}/../../.build/hamlib/4.7.2/darwin-amd64/include
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/../../.build/hamlib/4.7.2/darwin-amd64/lib/libhamlib.a
#cgo linux,amd64 CFLAGS: -I${SRCDIR}/../../.build/hamlib/4.7.2/linux-amd64/include
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/../../.build/hamlib/4.7.2/linux-amd64/lib/libhamlib.a -ldl -lm -pthread
#cgo windows,amd64 CFLAGS: -I${SRCDIR}/../../.build/hamlib/4.7.2/windows-amd64/include
#cgo windows,amd64 LDFLAGS: ${SRCDIR}/../../.build/hamlib/4.7.2/windows-amd64/lib/libhamlib.a -lws2_32 -liphlpapi -lwinmm

#include <hamlib/rig.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

typedef struct {
	int id;
	int default_baud_rate;
	char manufacturer[64];
	char name[96];
} ditdah_model;

typedef struct {
	ditdah_model *models;
	int capacity;
	int count;
} ditdah_model_collection;

static int ditdah_collect_model(const struct rig_caps *caps, void *opaque) {
	ditdah_model_collection *collection = (ditdah_model_collection *)opaque;
	if (caps->port_type != RIG_PORT_SERIAL || caps->get_freq == NULL) {
		return 1;
	}
	if (collection->count < collection->capacity) {
		ditdah_model *model = &collection->models[collection->count];
		model->id = caps->rig_model;
		model->default_baud_rate = caps->serial_rate_max;
		snprintf(model->manufacturer, sizeof(model->manufacturer), "%s", caps->mfg_name);
		snprintf(model->name, sizeof(model->name), "%s", caps->model_name);
	}
	collection->count++;
	return 1;
}

static int ditdah_list_models(ditdah_model *models, int capacity) {
	ditdah_model_collection collection = {models, capacity, 0};
	rig_set_debug(RIG_DEBUG_NONE);
	rig_load_all_backends();
	rig_list_foreach(ditdah_collect_model, &collection);
	return collection.count;
}

static int ditdah_set_conf(RIG *rig, const char *name, const char *value) {
	hamlib_token_t token = rig_token_lookup(rig, name);
	if (token == RIG_CONF_END) {
		return -RIG_EINVAL;
	}
	return rig_set_conf(rig, token, value);
}

static int ditdah_get_frequency(
	int model_id,
	const char *port,
	int baud_rate,
	unsigned long long *frequency_hz
) {
	rig_set_debug(RIG_DEBUG_NONE);
	RIG *rig = rig_init(model_id);
	if (rig == NULL) {
		return -RIG_EINVAL;
	}

	int result = ditdah_set_conf(rig, "rig_pathname", port);
	char baud[32];
	if (result == RIG_OK) {
		snprintf(baud, sizeof(baud), "%d", baud_rate);
		result = ditdah_set_conf(rig, "serial_speed", baud);
	}
	if (result == RIG_OK) {
		result = ditdah_set_conf(rig, "auto_power_on", "0");
	}
	if (result == RIG_OK) {
		result = ditdah_set_conf(rig, "auto_power_off", "0");
	}
	if (result == RIG_OK) {
		result = rig_open(rig);
	}
	if (result == RIG_OK) {
		freq_t frequency = 0;
		result = rig_get_freq(rig, RIG_VFO_CURR, &frequency);
		if (result == RIG_OK) {
			*frequency_hz = (unsigned long long)(frequency + 0.5);
		}
		int close_result = rig_close(rig);
		if (result == RIG_OK) {
			result = close_result;
		}
	}
	rig_cleanup(rig);
	return result;
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unsafe"

	serial "go.bug.st/serial"
)

type hamlibService struct {
	mu sync.Mutex
}

// New exposes the statically linked Hamlib implementation.
func New() Service {
	return &hamlibService{}
}

func (s *hamlibService) Models() ([]Model, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := int(C.ditdah_list_models(nil, 0))
	if count == 0 {
		return nil, errors.New("Hamlib did not return any serial radio models")
	}
	models := make([]C.ditdah_model, count)
	actual := int(C.ditdah_list_models(&models[0], C.int(len(models))))
	if actual < len(models) {
		models = models[:actual]
	}
	result := make([]Model, len(models))
	for index := range models {
		result[index] = Model{
			ID:              int(models[index].id),
			Manufacturer:    C.GoString(&models[index].manufacturer[0]),
			Name:            C.GoString(&models[index].name[0]),
			DefaultBaudRate: int(models[index].default_baud_rate),
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].Manufacturer + " " + result[i].Name
		right := result[j].Manufacturer + " " + result[j].Name
		return left < right
	})
	return result, nil
}

func (s *hamlibService) Ports() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("list serial ports: %w", err)
	}
	filtered := ports[:0]
	for _, port := range ports {
		if runtime.GOOS == "darwin" && strings.HasPrefix(port, "/dev/tty.") {
			continue
		}
		filtered = append(filtered, port)
	}
	sort.Strings(filtered)
	return filtered, nil
}

func (s *hamlibService) Check(
	ctx context.Context,
	config Config,
) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if config.ModelID <= 0 {
		return 0, errors.New("radio model is required")
	}
	if strings.TrimSpace(config.Port) == "" {
		return 0, errors.New("serial port is required")
	}
	if config.BaudRate <= 0 {
		return 0, errors.New("baud rate must be a positive number")
	}

	port := C.CString(config.Port)
	defer C.free(unsafe.Pointer(port))
	s.mu.Lock()
	defer s.mu.Unlock()
	var frequency C.ulonglong
	result := C.ditdah_get_frequency(
		C.int(config.ModelID),
		port,
		C.int(config.BaudRate),
		&frequency,
	)
	if result != C.RIG_OK {
		return 0, fmt.Errorf("radio connection failed: %s", C.GoString(C.rigerror(C.int(result))))
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return uint64(frequency), nil
}
