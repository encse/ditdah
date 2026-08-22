package tui

import (
	"context"
	"fmt"
	"sync"

	"morsemanual/internal/syncutil"
	"morsemanual/internal/tui/modal"

	"golang.org/x/sync/errgroup"
)

// layerRuntime coordinates requested layers with their running lifecycles.
// It owns the synchronization required by UI callbacks and the layer runner.
type layerRuntime struct {
	changes   syncutil.Mailbox[layerState]
	mu        sync.Mutex
	requested []requestedLayer
	running   map[modal.Owner]*runningLayer
}

type requestedLayer struct {
	parent modal.Owner
	page   Page
	modal  *openedModal
}

type layerState struct {
	layers        []requestedLayer
	stoppedPageID string
}

type runningLayer struct {
	requestedLayer
	ctx        context.Context
	cancel     context.CancelFunc
	group      *errgroup.Group
	taskMu     sync.Mutex
	stopping   bool
	lastUpdate <-chan struct{}
}

func newLayerRuntime() layerRuntime {
	return layerRuntime{running: make(map[modal.Owner]*runningLayer)}
}

func (r *layerRuntime) initialize(initial requestedLayer) {
	r.mu.Lock()
	r.changes = syncutil.NewMailbox(layerState{layers: []requestedLayer{initial}})
	r.requested = []requestedLayer{initial}
	r.running = make(map[modal.Owner]*runningLayer)
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
		if request.parent != nil &&
			(len(running) == 0 ||
				running[len(running)-1].identity() != request.parent) {
			break
		}
		layerCtx, cancel := context.WithCancel(parentCtx)
		group, runCtx := errgroup.WithContext(layerCtx)
		ready := make(chan struct{})
		close(ready)
		layer := &runningLayer{
			requestedLayer: request,
			ctx:            runCtx,
			cancel:         cancel,
			group:          group,
			lastUpdate:     ready,
		}
		r.register(layer)
		app.update(func() {
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
		r.unregister(running[index].identity())
	}
	app.update(func() {
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
	owner modal.Owner,
	requireOwner bool,
	opened *openedModal,
) {
	r.mu.Lock()
	if len(r.requested) == 0 {
		r.mu.Unlock()
		return
	}
	parent := r.requested[len(r.requested)-1]
	if requireOwner && parent.identity() != owner {
		r.mu.Unlock()
		return
	}
	r.requested = append(r.requested, requestedLayer{
		parent: parent.identity(),
		modal:  opened,
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
		r.requested[0].page != stopped.page {
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
	r.running[layer.identity()] = layer
	r.mu.Unlock()
}

func (r *layerRuntime) unregister(owner modal.Owner) {
	r.mu.Lock()
	delete(r.running, owner)
	r.mu.Unlock()
}

func (r *layerRuntime) requestIsCurrent(target requestedLayer) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, layer := range r.requested {
		if layer.identity() == target.identity() {
			return true
		}
	}
	return false
}

func (r *layerRuntime) startTask(
	owner modal.Owner,
	work func(*runningLayer) error,
) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	layer := r.runningOwnerLocked(owner)
	if layer == nil {
		return false
	}
	return layer.startTask(func() error { return work(layer) })
}

func (r *layerRuntime) startUpdate(
	owner modal.Owner,
	update func(*runningLayer) error,
) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	layer := r.runningOwnerLocked(owner)
	if layer == nil {
		return false
	}
	return layer.startUpdate(func() error { return update(layer) })
}

func (r *layerRuntime) runningOwnerLocked(
	owner modal.Owner,
) *runningLayer {
	for index := len(r.requested) - 1; index >= 0; index-- {
		requested := r.requested[index]
		if requested.identity() == owner {
			return r.running[owner]
		}
	}
	return nil
}

func (r *layerRuntime) isRequested(target *runningLayer) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, layer := range r.requested {
		if layer.identity() == target.identity() {
			return r.running[layer.identity()] == target
		}
	}
	return false
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

func (l *runningLayer) startUpdate(update func() error) bool {
	l.taskMu.Lock()
	defer l.taskMu.Unlock()
	if l.stopping || l.ctx.Err() != nil {
		return false
	}
	previous := l.lastUpdate
	done := make(chan struct{})
	l.lastUpdate = done
	l.group.Go(func() error {
		defer close(done)
		<-previous
		return update()
	})
	return true
}

func (l *runningLayer) stopTasks() {
	l.taskMu.Lock()
	l.stopping = true
	l.taskMu.Unlock()
}

func newPageLayer(page Page) requestedLayer {
	return requestedLayer{page: page}
}

func (r requestedLayer) identity() modal.Owner {
	if r.modal != nil {
		return r.modal.dialog
	}
	return r.page
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
		if running[index].identity() != requested[index].identity() {
			return index
		}
	}
	return limit
}
