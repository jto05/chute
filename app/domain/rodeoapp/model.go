package rodeoapp

// SyncResult summarizes the outcome of a sync run.
type SyncResult struct {
	Fetched int
	Stored  int
	Skipped int // already up-to-date
}
