package plugin

import (
	"errors"
	"seanime/internal/extension"
	"seanime/internal/library/playbackmanager"
	goja_util "seanime/internal/util/goja"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Playback struct {
	ctx       *AppContextImpl
	vm        *goja.Runtime
	logger    *zerolog.Logger
	ext       *extension.Extension
	scheduler *goja_util.Scheduler
}

func (a *AppContextImpl) BindPlaybackToContextObj(vm *goja.Runtime, obj *goja.Object, logger *zerolog.Logger, ext *extension.Extension, scheduler *goja_util.Scheduler) {
	p := &Playback{
		ctx:       a,
		vm:        vm,
		logger:    logger,
		ext:       ext,
		scheduler: scheduler,
	}

	playbackObj := vm.NewObject()
	_ = playbackObj.Set("registerEventListener", p.registerEventListener)
	_ = playbackObj.Set("getNextEpisode", p.getNextEpisode)
	_ = obj.Set("playback", playbackObj)
}

type PlaybackEvent struct {
	IsVideoStarted    bool `json:"isVideoStarted"`
	IsVideoStopped    bool `json:"isVideoStopped"`
	IsVideoCompleted  bool `json:"isVideoCompleted"`
	IsStreamStarted   bool `json:"isStreamStarted"`
	IsStreamStopped   bool `json:"isStreamStopped"`
	IsStreamCompleted bool `json:"isStreamCompleted"`
	StartedEvent      *struct {
		Filename string `json:"filename"`
	} `json:"startedEvent"`
	StoppedEvent *struct {
		Reason string `json:"reason"`
	} `json:"stoppedEvent"`
	CompletedEvent *struct {
		Filename string `json:"filename"`
	} `json:"completedEvent"`
	State *playbackmanager.PlaybackState `json:"state"`
}

// registerEventListener registers a subscriber for playback events.
func (p *Playback) registerEventListener(callback func(event *PlaybackEvent)) (func(), error) {
	playbackManager, ok := p.ctx.PlaybackManager().Get()
	if !ok {
		return nil, errors.New("playback manager not found")
	}

	id := uuid.New().String()

	subscriber := playbackManager.SubscribeToPlaybackStatus(id)

	go func() {
		for event := range subscriber.EventCh {
			switch e := event.(type) {
			case playbackmanager.PlaybackStatusChangedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						State: &e.State,
					})
					return nil
				})
			case playbackmanager.VideoStartedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						IsVideoStarted: true,
						StartedEvent: &struct {
							Filename string `json:"filename"`
						}{
							Filename: e.Filename,
						},
					})
					return nil
				})
			case playbackmanager.VideoStoppedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						IsVideoStopped: true,
						StoppedEvent: &struct {
							Reason string `json:"reason"`
						}{
							Reason: e.Reason,
						},
					})
					return nil
				})
			case playbackmanager.VideoCompletedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						IsVideoCompleted: true,
						CompletedEvent: &struct {
							Filename string `json:"filename"`
						}{
							Filename: e.Filename,
						},
					})
					return nil
				})
			case playbackmanager.StreamStateChangedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						State: &e.State,
					})
					return nil
				})
			case playbackmanager.StreamStartedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						IsStreamStarted: true,
						StartedEvent: &struct {
							Filename string `json:"filename"`
						}{
							Filename: e.Filename,
						},
					})
					return nil
				})
			case playbackmanager.StreamStoppedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						IsStreamStopped: true,
						StoppedEvent: &struct {
							Reason string `json:"reason"`
						}{
							Reason: e.Reason,
						},
					})
					return nil
				})
			case playbackmanager.StreamCompletedEvent:
				p.scheduler.ScheduleAsync(func() error {
					callback(&PlaybackEvent{
						IsStreamCompleted: true,
						CompletedEvent: &struct {
							Filename string `json:"filename"`
						}{
							Filename: e.Filename,
						},
					})
					return nil
				})
			}
		}
	}()

	cancelFn := func() {
		playbackManager.UnsubscribeFromPlaybackStatus(id)
	}

	return cancelFn, nil
}

func (p *Playback) getNextEpisode() goja.Value {
	promise, resolve, reject := p.vm.NewPromise()

	playbackManager, ok := p.ctx.PlaybackManager().Get()
	if !ok {
		reject(p.vm.NewGoError(errors.New("playback manager not found")))
		return p.vm.ToValue(promise)
	}

	go func() {
		nextEpisode := playbackManager.GetNextEpisode()
		p.scheduler.ScheduleAsync(func() error {
			resolve(p.vm.ToValue(nextEpisode))
			return nil
		})
	}()
	return p.vm.ToValue(promise)
}
