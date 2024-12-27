package state

func StringPtr(from map[string]interface{}, key string, defaultval *string) *string {
	return SafeMapValue(from, key, defaultval)
}

func MustStringPtr(from map[string]any, key string) *string {
	return MustMapValue[string, any, *string](from, key)
}

func String(from map[string]interface{}, key string, defaultval string) string {
	return SafeMapValue(from, key, defaultval)
}

func MustString(from map[string]interface{}, key string) string {
	return MustMapValue[string, interface{}, string](from, key)
}

func Int(from map[string]interface{}, key string, defaultval int) int {
	return SafeMapValue(from, key, defaultval)
}

func MustInt(from map[string]interface{}, key string) int {
	return MustMapValue[string, interface{}, int](from, key)
}

func Bool(from map[string]interface{}, key string, defaultval bool) bool {
	return SafeMapValue(from, key, defaultval)
}

func MustBool(from map[string]interface{}, key string) bool {
	return MustMapValue[string, interface{}, bool](from, key)
}

func MapStringInterface(from map[string]interface{}, key string, defaultval map[string]interface{}) map[string]interface{} {
	return SafeMapValue(from, key, defaultval)
}

func MapStringInterfaceSlice(from map[string]interface{}, key string, defaultval []map[string]interface{}) []map[string]interface{} {
	return SafeMapValue(from, key, defaultval)
}

func SafeMapValue[K comparable, T any, V any](from map[K]T, key K, defaultval V) V {
	if val, ok := from[key]; ok {
		if v, ok := any(val).(V); ok {
			return v
		}
	}
	return defaultval
}

func MustMapValue[K comparable, T any, V any](from map[K]T, key K) V {
	return any(from[key]).(V)
}
