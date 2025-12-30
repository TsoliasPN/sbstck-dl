package cmd

import "time"

type DownloadEventType string

const (
	DownloadEventStart       DownloadEventType = "start"
	DownloadEventPlan        DownloadEventType = "plan"
	DownloadEventPostStart   DownloadEventType = "post_start"
	DownloadEventPostDone    DownloadEventType = "post_done"
	DownloadEventPostFailed  DownloadEventType = "post_failed"
	DownloadEventPostSkipped DownloadEventType = "post_skipped"
	DownloadEventRetry       DownloadEventType = "retry"
	DownloadEventSummary     DownloadEventType = "summary"
)

type DownloadEvent struct {
	Type       DownloadEventType
	URL        string
	Path       string
	Title      string
	Reason     string
	Error      string
	Total      int
	Skipped    int
	Refreshed  int
	Downloaded int
	Failed     int
	RetryCount int
	RetryWait  time.Duration
}

type DownloadObserver func(DownloadEvent)

func emitDownloadEvent(observer DownloadObserver, event DownloadEvent) {
	if observer == nil {
		return
	}
	observer(event)
}
