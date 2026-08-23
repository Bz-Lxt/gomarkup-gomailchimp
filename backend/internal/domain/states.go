package domain

// Campaign lifecycle: draft → scheduled → running → paused → completed | failed | cancelled
var campaignTransitions = map[string][]string{
	"draft":     {"scheduled", "running", "cancelled"},
	"scheduled": {"running", "cancelled", "draft"},
	"running":   {"paused", "completed", "failed", "cancelled"},
	"paused":    {"running", "cancelled"},
	"completed": {},
	"failed":    {"running"},
	"cancelled": {},
}

func CanTransitCampaign(from, to string) bool {
	for _, n := range campaignTransitions[from] {
		if n == to {
			return true
		}
	}
	return false
}

// Recipient: pending → queued → sending → sent → delivered | bounced | failed | skipped
var recipientTransitions = map[string][]string{
	"pending":   {"queued", "skipped"},
	"queued":    {"sending", "skipped", "pending"},
	"sending":   {"sent", "failed", "bounced", "queued"},
	"sent":      {"delivered", "bounced"},
	"delivered": {"bounced"},
	"bounced":   {},
	"failed":    {"queued"},
	"skipped":   {},
}

func CanTransitRecipient(from, to string) bool {
	for _, n := range recipientTransitions[from] {
		if n == to {
			return true
		}
	}
	return false
}

// Contact subscribe machine
var contactTransitions = map[string][]string{
	"subscribed":   {"unsubscribed", "bounced", "complained", "suppressed"},
	"unsubscribed": {"subscribed"},
	"bounced":      {"suppressed", "subscribed"},
	"complained":   {"suppressed"},
	"suppressed":   {"subscribed"},
}

func CanTransitContact(from, to string) bool {
	for _, n := range contactTransitions[from] {
		if n == to {
			return true
		}
	}
	return false
}
