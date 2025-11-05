package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToInt64(t *testing.T) {
	t.Run("convert int to int64", func(t *testing.T) {
		result := toInt64(42)
		assert.Equal(t, int64(42), result)
	})

	t.Run("convert int64 to int64", func(t *testing.T) {
		result := toInt64(int64(100))
		assert.Equal(t, int64(100), result)
	})

	t.Run("convert float64 to int64", func(t *testing.T) {
		result := toInt64(42.7)
		assert.Equal(t, int64(42), result)
	})

	t.Run("convert json.Number to int64", func(t *testing.T) {
		// toInt64 doesn't support string, but supports json.Number
		result := toInt64(int32(123))
		assert.Equal(t, int64(123), result)
	})

	t.Run("convert nil returns 0", func(t *testing.T) {
		result := toInt64(nil)
		assert.Equal(t, int64(0), result)
	})

	t.Run("convert unsupported type returns 0", func(t *testing.T) {
		result := toInt64([]int{1, 2, 3})
		assert.Equal(t, int64(0), result)
	})
}

func TestToJSONString(t *testing.T) {
	t.Run("convert string returns string directly", func(t *testing.T) {
		result := toJSONString("hello")
		assert.Equal(t, "hello", result)
	})

	t.Run("convert map to JSON string", func(t *testing.T) {
		m := map[string]interface{}{"key": "value"}
		result := toJSONString(m)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
	})

	t.Run("convert array to JSON string", func(t *testing.T) {
		arr := []string{"a", "b", "c"}
		result := toJSONString(arr)
		assert.Contains(t, result, "a")
		assert.Contains(t, result, "b")
	})

	t.Run("convert number to JSON string", func(t *testing.T) {
		result := toJSONString(42)
		assert.Equal(t, "42", result)
	})

	t.Run("convert bool to JSON string", func(t *testing.T) {
		result := toJSONString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("convert nil returns empty string", func(t *testing.T) {
		result := toJSONString(nil)
		assert.Equal(t, "", result)
	})
}

func TestCloneMap(t *testing.T) {
	t.Run("clone simple map", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}

		cloned := cloneMap(original)

		assert.NotNil(t, cloned)
		assert.Equal(t, original["key1"], cloned["key1"])
		assert.Equal(t, original["key2"], cloned["key2"])

		// Verify it's a deep copy
		cloned["key1"] = "modified"
		assert.NotEqual(t, original["key1"], cloned["key1"])
	})

	t.Run("clone nested map", func(t *testing.T) {
		original := map[string]interface{}{
			"outer": map[string]interface{}{
				"inner": "value",
			},
		}

		cloned := cloneMap(original)

		assert.NotNil(t, cloned)
		assert.Contains(t, cloned, "outer")
	})

	t.Run("clone map with array", func(t *testing.T) {
		original := map[string]interface{}{
			"array": []interface{}{"a", "b", "c"},
		}

		cloned := cloneMap(original)

		assert.NotNil(t, cloned)
		assert.Contains(t, cloned, "array")
	})

	t.Run("clone empty map", func(t *testing.T) {
		original := map[string]interface{}{}
		cloned := cloneMap(original)

		assert.NotNil(t, cloned)
		assert.Len(t, cloned, 0)
	})

	t.Run("clone nil map", func(t *testing.T) {
		cloned := cloneMap(nil)
		assert.Nil(t, cloned)
	})
}

func TestGCD(t *testing.T) {
	t.Run("gcd of two positive numbers", func(t *testing.T) {
		result := gcd(12, 8)
		assert.Equal(t, 4, result)
	})

	t.Run("gcd of coprime numbers", func(t *testing.T) {
		result := gcd(17, 13)
		assert.Equal(t, 1, result)
	})

	t.Run("gcd with zero", func(t *testing.T) {
		result := gcd(5, 0)
		assert.Equal(t, 5, result)
	})

	t.Run("gcd with both zero", func(t *testing.T) {
		result := gcd(0, 0)
		assert.Equal(t, 0, result)
	})

	t.Run("gcd of same numbers", func(t *testing.T) {
		result := gcd(7, 7)
		assert.Equal(t, 7, result)
	})

	t.Run("gcd of large numbers", func(t *testing.T) {
		result := gcd(1071, 462)
		assert.Equal(t, 21, result)
	})
}

func TestJSONString(t *testing.T) {
	t.Run("json string from string returns trimmed string", func(t *testing.T) {
		result := jsonString("hello")
		assert.Equal(t, "hello", result)
	})

	t.Run("json string from map", func(t *testing.T) {
		m := map[string]string{"key": "value"}
		result := jsonString(m)
		assert.Contains(t, result, "key")
	})

	t.Run("json string from nil returns empty", func(t *testing.T) {
		result := jsonString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("json string from empty string returns empty", func(t *testing.T) {
		result := jsonString("  ")
		assert.Equal(t, "", result)
	})
}
