package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseLocation parses a timezone string into a time.Location.
// Supports IANA names (e.g. "Asia/Bangkok") and fixed offsets like "UTC+7" or "UTC-03:30".
func ParseLocation(tz string) (*time.Location, error) {
	trimmed := strings.TrimSpace(tz)
	if trimmed == "" {
		return time.FixedZone("UTC", 0), nil
	}

	upper := strings.ToUpper(trimmed)
	if upper == "UTC" || upper == "GMT" {
		return time.UTC, nil
	}

	if strings.HasPrefix(upper, "UTC") || strings.HasPrefix(upper, "GMT") {
		offset := strings.TrimPrefix(strings.TrimPrefix(upper, "UTC"), "GMT")
		if offset == "" {
			return time.UTC, nil
		}
		sign := 1
		switch offset[0] {
		case '+':
			offset = offset[1:]
		case '-':
			sign = -1
			offset = offset[1:]
		default:
			return nil, fmt.Errorf("invalid UTC offset format: %q", tz)
		}

		hours := 0
		minutes := 0

		if strings.Contains(offset, ":") {
			parts := strings.SplitN(offset, ":", 2)
			var err error
			hours, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid UTC offset hour in %q: %w", tz, err)
			}
			minutes, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid UTC offset minute in %q: %w", tz, err)
			}
		} else if len(offset) == 4 {
			// Support forms like "0530"
			h, err := strconv.Atoi(offset[:2])
			if err != nil {
				return nil, fmt.Errorf("invalid UTC offset hour in %q: %w", tz, err)
			}
			m, err := strconv.Atoi(offset[2:])
			if err != nil {
				return nil, fmt.Errorf("invalid UTC offset minute in %q: %w", tz, err)
			}
			hours = h
			minutes = m
		} else {
			h, err := strconv.Atoi(offset)
			if err != nil {
				return nil, fmt.Errorf("invalid UTC offset hour in %q: %w", tz, err)
			}
			hours = h
		}

		if minutes < 0 || minutes >= 60 {
			return nil, fmt.Errorf("invalid UTC offset minute in %q", tz)
		}

		offsetSeconds := sign * (hours*3600 + minutes*60)
		return time.FixedZone(fmt.Sprintf("UTC%+02d:%02d", sign*hours, minutes), offsetSeconds), nil
	}

	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, err
	}
	return loc, nil
}
