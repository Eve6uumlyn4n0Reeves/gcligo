package config

import (
	"reflect"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	legacyFieldMappings map[string]string
	legacyMappingOnce   sync.Once
	legacyWarned        sync.Map
)

func (c *Config) warnLegacyOverrides(context string) {
	if c == nil {
		return
	}
	ensureLegacyMappings()
	cfgVal := reflect.ValueOf(c)
	if cfgVal.Kind() != reflect.Pointer {
		return
	}
	cfgVal = cfgVal.Elem()

	for legacyField, domainPath := range legacyFieldMappings {
		legacyValue := cfgVal.FieldByName(legacyField)
		if !legacyValue.IsValid() || isZeroValue(legacyValue) {
			continue
		}
		domainValue := fieldByPath(cfgVal, domainPath)
		if !domainValue.IsValid() || !reflect.DeepEqual(legacyValue.Interface(), domainValue.Interface()) {
			warnLegacyField(legacyField, domainPath, context)
		}
	}
}

func warnLegacyField(field, domainPath, context string) {
	if _, loaded := legacyWarned.LoadOrStore(field, struct{}{}); loaded {
		return
	}
	log.WithFields(log.Fields{
		"legacy_field": field,
		"replacement":  domainPath,
		"context":      context,
	}).Warn("legacy config field is still in use; migrate to domain struct")
}

func ensureLegacyMappings() {
	legacyMappingOnce.Do(func() {
		legacyFieldMappings = buildLegacyMappings()
	})
}

func buildLegacyMappings() map[string]string {
	cfgType := reflect.TypeOf(Config{})
	simpleFields := make(map[string]struct{})
	for i := 0; i < cfgType.NumField(); i++ {
		field := cfgType.Field(i)
		if isDomainStruct(field.Type) {
			continue
		}
		simpleFields[field.Name] = struct{}{}
	}

	domainTypes := map[string]reflect.Type{
		"Server":          reflect.TypeOf(ServerConfig{}),
		"Upstream":        reflect.TypeOf(UpstreamConfig{}),
		"Security":        reflect.TypeOf(SecurityConfig{}),
		"Execution":       reflect.TypeOf(ExecutionConfig{}),
		"Storage":         reflect.TypeOf(StorageConfig{}),
		"Retry":           reflect.TypeOf(RetryConfig{}),
		"RateLimit":       reflect.TypeOf(RateLimitConfig{}),
		"APICompat":       reflect.TypeOf(APICompatConfig{}),
		"ResponseShaping": reflect.TypeOf(ResponseShapingConfig{}),
		"OAuth":           reflect.TypeOf(OAuthConfig{}),
		"AutoBan":         reflect.TypeOf(AutoBanConfig{}),
		"AutoProbe":       reflect.TypeOf(AutoProbeConfig{}),
		"Routing":         reflect.TypeOf(RoutingConfig{}),
	}

	mapping := make(map[string]string)
	for domainName, typ := range domainTypes {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			for _, alias := range candidateNames(domainName, field.Name) {
				if _, ok := simpleFields[alias]; ok {
					mapping[alias] = domainName + "." + field.Name
					delete(simpleFields, alias)
					break
				}
			}
		}
	}
	return mapping
}

func candidateNames(domainName, fieldName string) []string {
	base := strings.TrimSuffix(domainName, "Config")
	aliases := []string{
		domainName + fieldName,
		base + fieldName,
		fieldName,
	}

	if base == "AutoBan" && strings.HasPrefix(fieldName, "Ban") {
		aliases = append(aliases, base+strings.TrimPrefix(fieldName, "Ban"))
	}
	if base == "AutoBan" && strings.HasPrefix(fieldName, "Recovery") {
		aliases = append(aliases, "Auto"+fieldName)
	}
	if base == "Routing" && strings.HasPrefix(fieldName, "Cooldown") {
		aliases = append(aliases, "Router"+fieldName)
	}
	if base == "Routing" && fieldName == "PersistState" {
		aliases = append(aliases, "PersistRoutingState")
	}
	if base == "Routing" && fieldName == "PersistIntervalSec" {
		aliases = append(aliases, "RoutingPersistIntervalSec")
	}
	if base == "ResponseShaping" && strings.HasPrefix(fieldName, "FakeStreaming") {
		aliases = append(aliases, fieldName)
	}
	if base == "OAuth" && strings.HasPrefix(fieldName, "Refresh") {
		aliases = append(aliases, fieldName)
	}
	return dedupe(aliases)
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func isDomainStruct(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	switch t {
	case reflect.TypeOf(ServerConfig{}),
		reflect.TypeOf(UpstreamConfig{}),
		reflect.TypeOf(SecurityConfig{}),
		reflect.TypeOf(ExecutionConfig{}),
		reflect.TypeOf(StorageConfig{}),
		reflect.TypeOf(RetryConfig{}),
		reflect.TypeOf(RateLimitConfig{}),
		reflect.TypeOf(APICompatConfig{}),
		reflect.TypeOf(ResponseShapingConfig{}),
		reflect.TypeOf(OAuthConfig{}),
		reflect.TypeOf(AutoBanConfig{}),
		reflect.TypeOf(AutoProbeConfig{}),
		reflect.TypeOf(RoutingConfig{}):
		return true
	default:
		return false
	}
}

func fieldByPath(root reflect.Value, path string) reflect.Value {
	current := root
	if current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	for _, part := range strings.Split(path, ".") {
		if !current.IsValid() {
			return reflect.Value{}
		}
		if current.Kind() == reflect.Pointer {
			current = current.Elem()
		}
		current = current.FieldByName(part)
	}
	return current
}

func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	return v.IsZero()
}

// resetLegacyWarnings is a test helper to reset warning state.
func resetLegacyWarnings() {
	legacyWarned = sync.Map{}
}
