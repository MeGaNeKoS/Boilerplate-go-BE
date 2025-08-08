package system

// LongInput is used by the Long handler to delay responses.
type LongInput struct {
	Sleep int `query:"sleep" doc:"Seconds to sleep" example:"1"`
}
