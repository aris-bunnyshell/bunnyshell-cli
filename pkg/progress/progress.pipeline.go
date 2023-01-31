package progress

import (
	"fmt"
	"time"

	"bunnyshell.com/sdk"
	"github.com/briandowns/spinner"
)

type PipelineSyncer func() (*sdk.PipelineItem, error)

type Progress struct {
	Options Options

	spinner *spinner.Spinner

	stages map[string]bool
}

type Options struct {
	Interval time.Duration
}

func NewOptions() *Options {
	return &Options{
		Interval: defaultUpdate,
	}
}

func NewPipeline(options Options) *Progress {
	spinner := newSpinner()
	spinner.Prefix = fmt.Sprintf(
		"%s Fetching pipeline status... ",
		prefixWait,
	)

	return &Progress{
		Options: options,

		spinner: spinner,
		stages:  map[string]bool{},
	}
}

func (p *Progress) Update(pipelineSync PipelineSyncer) error {
	for {
		pipeline, err := pipelineSync()
		if err != nil {
			return err
		}

		if pipeline == nil {
			return nil
		}

		if !p.UpdatePipeline(pipeline) {
			return nil
		}

		time.Sleep(p.Options.Interval)
	}
}

func (p *Progress) UpdatePipeline(pipeline *sdk.PipelineItem) InProgress {
	if pipeline == nil {
		return false
	}

	p.spinner.Prefix = "Processing Pipeline "

	for _, stage := range pipeline.GetStages() {
		switch p.setStage(stage) {
		case Success:
			continue
		case Failed:
			return false
		case Synced:
			return true
		}
	}

	return false
}

func (p *Progress) Start() {
	p.spinner.Start()
}

func (p *Progress) Stop() {
	p.spinner.Stop()
}

func (p *Progress) setStage(stage sdk.StageItem) UpdateStatus {
	if stage.GetStatus() == "failed" {
		p.finishStage(stage)

		return Failed
	}

	if stage.GetStatus() == "success" {
		p.finishStage(stage)

		return Success
	}

	p.syncStage(stage)

	return Synced
}

func (p *Progress) finishStage(stage sdk.StageItem) {
	if p.stages[stage.GetId()] {
		return
	}

	p.stages[stage.GetId()] = true

	prefix := p.getPrefix(stage)

	p.spinner.FinalMSG = fmt.Sprintf(
		"%s %s finished %d jobs in %s\n",
		prefix,
		stage.GetName(),
		stage.GetJobsCount(),
		time.Duration(stage.GetDuration())*time.Second,
	)

	p.spinner.Restart()

	p.spinner.FinalMSG = ""
}

func (p *Progress) syncStage(stage sdk.StageItem) {
	p.spinner.Prefix = fmt.Sprintf(
		"%s %s... %d/%d jobs completed ",
		p.getPrefix(stage),
		stage.GetName(),
		stage.GetCompletedJobsCount(),
		stage.GetJobsCount(),
	)
}

func (p *Progress) getPrefix(stage sdk.StageItem) string {
	switch stage.GetStatus() {
	case "success":
		return prefixDone
	case "in_progress":
		return prefixWait
	case "pending":
		return prefixWait
	case "failed":
		return prefixErr
	}

	return prefixUnk
}
