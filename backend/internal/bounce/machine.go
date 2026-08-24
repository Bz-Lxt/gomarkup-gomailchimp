package bounce

import "github.com/lumen/relay/internal/domain"

type Action struct {
	ContactStatus string
	Suppress      bool
	Reason        string
	IncrementSoft bool
}

const SoftThreshold = 3

func Next(currentStatus string, ev domain.BounceEvent, softCount int) Action {
	switch ev.Class {
	case domain.BounceHard:
		return Action{ContactStatus: "suppressed", Suppress: true, Reason: "hard_bounce"}
	case domain.BounceBlock:
		if ev.Code == "535" {
			return Action{} // auth failure is channel-level, not recipient
		}
		return Action{ContactStatus: "suppressed", Suppress: true, Reason: "block"}
	case domain.BounceSoft:
		if softCount+1 >= SoftThreshold {
			return Action{ContactStatus: "suppressed", Suppress: true, Reason: "soft_bounce_threshold", IncrementSoft: true}
		}
		return Action{ContactStatus: currentStatus, IncrementSoft: true, Reason: "soft_bounce"}
	}
	return Action{}
}

func OnComplaint() Action {
	return Action{ContactStatus: "complained", Suppress: true, Reason: "complaint"}
}

func OnUnsubscribe() Action {
	return Action{ContactStatus: "unsubscribed", Suppress: true, Reason: "unsubscribe"}
}
