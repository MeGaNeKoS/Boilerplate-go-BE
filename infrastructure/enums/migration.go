package enums

// DirtyStrategy defines how to handle a dirty migration state.
type DirtyStrategy string

const (
	DirtyStrategyRetry DirtyStrategy = "retry"
	DirtyStrategySkip  DirtyStrategy = "skip"
)
