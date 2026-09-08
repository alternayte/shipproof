package src

// Allow reports whether a caller may make one more request. It rejects the
// caller once the count reaches the limit.
func Allow(count, limit int) bool {
	return count < limit
}
