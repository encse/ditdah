package tui

import (
	"context"
	"fmt"
	"sync"

	"morsemanual/internal/mailbox"

	"github.com/rivo/tview"
	"golang.org/x/sync/errgroup"
)

// layerRuntime coordinates requested layers with their running lifecycles.
// It owns the synchronization required by UI callbacks and the layer runner.
type layerRuntime struct {
	changes   mailbox.Mailbox[layerState]
	mu        sync.Mutex
	requested []requestedLayer
	running   map[*layerInstance]*runningLayer
}

type requestedLayer struct {
	instance *layerInstance
	owner    *layerInstance
	page     Page
	modal    *openedModal
}

// layerInstance is deliberately non-zero-sized so distinct allocations always
// have distinct pointer identity.
type layerInstance struct {
	identity byte
}

type layerState struct {
	layers        []requestedLayer
	stoppedPageID string
}

type runningLayer struct {
	requestedLayer
	ctx      context.Context
	cancel   context.CancelFunc
	group    *errgroup.Group
	taskMu   sync.Mutex
	stopping bool
}

func newLayerRuntime() layerRuntime {
	return layerRuntime{running: make(map[*layerInstance]*runningLayer)}
}

func (r *layerRuntime) initialize(initial requestedLayer) {
	r.mu.Lock()
	r.changes = mailbox.New(layerState{layers: []requestedLayer{initial}})
	r.requested = []requestedLayer{initial}
	r.running = make(map[*layerInstance]*runningLayer)
	r.mu.Unlock()
}

func (r *layerRuntime) receive(ctx context.Context) (layerState, error) {
	return r.changes.Receive(ctx)
}

// Run reconciles requested pages and modals with their running lifecycles.
func (r *layerRuntime) Run(ctx context.Context, app *application) error {
	var running []*runningLayer
	for {
		state, err := r.receive(ctx)
		if err != nil {
			r.stop(running, 0, app)
			return nil
		}

		common := commonLayerPrefix(running, state.layers)
		r.stop(running, common, app)
		running = running[:common]
		running = r.start(ctx, running, state.layers[common:], app)

		if state.stoppedPageID != "" {
			r.stop(running, 0, app)
			return fmt.Errorf(
				"run page %q: %w",
				state.stoppedPageID,
				errPageStopped,
			)
		}
	}
}

func (r *layerRuntime) start(
	rootCtx context.Context,
	running []*runningLayer,
	requested []requestedLayer,
	app *application,
) []*runningLayer {
	parentCtx := rootCtx
	if len(running) > 0 {
		parentCtx = running[len(running)-1].ctx
	}
	for _, request := range requested {
		if !r.requestIsCurrent(request) {
			break
		}
		if request.owner != nil &&
			(len(running) == 0 ||
				running[len(running)-1].instance != request.owner) {
			break
		}
		layerCtx, cancel := context.WithCancel(parentCtx)
		group, runCtx := errgroup.WithContext(layerCtx)
		layer := &runningLayer{
			requestedLayer: request,
			ctx:            runCtx,
			cancel:         cancel,
			group:          group,
		}
		r.register(layer)
		app.Update(func() {
			if request.modal != nil {
				app.showModal(request.modal)
				return
			}
			app.showPage(request.page)
		})
		group.Go(func() error {
			request.run(runCtx)
			if runCtx.Err() == nil {
				r.layerReturned(request)
			}
			return nil
		})
		running = append(running, layer)
		parentCtx = runCtx
	}
	return running
}

func (r *layerRuntime) stop(
	running []*runningLayer,
	from int,
	app *application,
) {
	if from >= len(running) {
		return
	}
	for _, layer := range running[from:] {
		layer.stopTasks()
		layer.cancel()
	}
	for index := len(running) - 1; index >= from; index-- {
		_ = running[index].group.Wait()
		r.unregister(running[index].instance)
	}
	app.Update(func() {
		for layerIndex := len(running) - 1; layerIndex >= from; layerIndex-- {
			opened := running[layerIndex].modal
			if opened == nil {
				continue
			}
			for modalIndex := len(app.modals) - 1; modalIndex >= 0; modalIndex-- {
				if app.modals[modalIndex] != opened {
					continue
				}
				if opened.overlay != nil {
					opened.overlay.Close()
				}
				app.modals = append(
					app.modals[:modalIndex],
					app.modals[modalIndex+1:]...,
				)
				break
			}
		}
	})
}

func (r *layerRuntime) layerReturned(layer requestedLayer) {
	if layer.modal != nil {
		r.requestModalClose(layer.modal)
		return
	}
	r.requestLayerStopped(layer)
}

func (r *layerRuntime) requestPage(page Page) {
	r.mu.Lock()
	r.requested = []requestedLayer{newPageLayer(page)}
	state := r.stateLocked("")
	r.mu.Unlock()
	r.changes.Send(state)
}

func (r *layerRuntime) requestModal(
	owner tview.Primitive,
	requireOwner bool,
	opened *openedModal,
) {
	r.mu.Lock()
	if len(r.requested) == 0 {
		r.mu.Unlock()
		return
	}
	parent := r.requested[len(r.requested)-1]
	if requireOwner && parent.content() != owner {
		r.mu.Unlock()
		return
	}
	r.requested = append(r.requested, requestedLayer{
		instance: &layerInstance{},
		owner:    parent.instance,
		modal:    opened,
	})
	state := r.stateLocked("")
	r.mu.Unlock()
	r.changes.Send(state)
}

func (r *layerRuntime) requestModalClose(target *openedModal) {
	r.mu.Lock()
	index := -1
	for layerIndex, layer := range r.requested {
		if layer.modal == target {
			index = layerIndex
			break
		}
	}
	if index < 0 {
		r.mu.Unlock()
		return
	}
	r.requested = r.requested[:index]
	state := r.stateLocked("")
	r.mu.Unlock()
	r.changes.Send(state)
}

func (r *layerRuntime) requestLayerStopped(stopped requestedLayer) {
	r.mu.Lock()
	if len(r.requested) == 0 || r.requested[0].page == nil ||
		r.requested[0].instance != stopped.instance {
		r.mu.Unlock()
		return
	}
	r.requested = nil
	state := r.stateLocked(stopped.page.ID())
	r.mu.Unlock()
	r.changes.Send(state)
}

func (r *layerRuntime) register(layer *runningLayer) {
	r.mu.Lock()
	r.running[layer.instance] = layer
	r.mu.Unlock()
}

func (r *layerRuntime) unregister(instance *layerInstance) {
	r.mu.Lock()
	delete(r.running, instance)
	r.mu.Unlock()
}

func (r *layerRuntime) requestIsCurrent(target requestedLayer) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, layer := range r.requested {
		if layer.instance == target.instance {
			return true
		}
	}
	return false
}

func (r *layerRuntime) startBackground(
	owner tview.Primitive,
	work func(*runningLayer) error,
) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requested) == 0 {
		return false
	}
	requested := r.requested[len(r.requested)-1]
	if requested.content() != owner {
		return false
	}
	layer := r.running[requested.instance]
	if layer == nil {
		return false
	}
	return layer.startTask(func() error { return work(layer) })
}

func (r *layerRuntime) topIs(instance *layerInstance) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requested) > 0 &&
		r.requested[len(r.requested)-1].instance == instance
}

func (r *layerRuntime) requestedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requested)
}

func (r *layerRuntime) stateLocked(stoppedPageID string) layerState {
	return layerState{
		layers:        append([]requestedLayer(nil), r.requested...),
		stoppedPageID: stoppedPageID,
	}
}

func (l *runningLayer) startTask(work func() error) bool {
	l.taskMu.Lock()
	defer l.taskMu.Unlock()
	if l.stopping || l.ctx.Err() != nil {
		return false
	}
	l.group.Go(work)
	return true
}

func (l *runningLayer) stopTasks() {
	l.taskMu.Lock()
	l.stopping = true
	l.taskMu.Unlock()
}

func newPageLayer(page Page) requestedLayer {
	return requestedLayer{instance: &layerInstance{}, page: page}
}

func (r requestedLayer) content() tview.Primitive {
	if r.modal != nil {
		return r.modal.dialog.Content()
	}
	return r.page.Content()
}

func (r requestedLayer) run(ctx context.Context) {
	if r.modal != nil {
		r.modal.dialog.Run(ctx)
		return
	}
	r.page.Run(ctx)
}

func commonLayerPrefix(
	running []*runningLayer,
	requested []requestedLayer,
) int {
	limit := min(len(running), len(requested))
	for index := 0; index < limit; index++ {
		if running[index].instance != requested[index].instance {
			return index
		}
	}
	return limit
}
