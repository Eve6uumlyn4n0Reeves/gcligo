package strategy

import "sync"

var (
	defaultOnce  sync.Once
	defaultStrat *Strategy
)

// SetDefault sets the global default strategy if not already set.
func SetDefault(s *Strategy) {
	if s == nil {
		return
	}
	defaultOnce.Do(func() { defaultStrat = s })
}

// GetDefault returns the global default strategy if set.
func GetDefault() *Strategy { return defaultStrat }
