package sigma

// Event is a flat telemetry record. Field names are case-sensitive; ordinary
// values are case-insensitive, while regexes preserve case and escape semantics.
type Event map[string]any

// Match validates and evaluates a rule. For streams, Compile once and reuse it.
func (r Rule) Match(ev Event) (bool, error) {
	c, err := r.Compile()
	if err != nil {
		return false, err
	}
	return c.Match(ev), nil
}
