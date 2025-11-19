package testlogger_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ashep/go-app/testlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufWriter_Write(t *testing.T) {
	t.Run("WritesValidNestedJSON", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		// Create a nested JSON structure like zerolog would produce
		innerJSON := `{"level":"info","message":"test message","time":"2023-01-01T00:00:00Z"}`
		outerJSON := `{"message":"` + strings.ReplaceAll(innerJSON, `"`, `\"`) + `"}`

		n, err := writer.Write([]byte(outerJSON))

		require.NoError(t, err)
		assert.Greater(t, n, 0)

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Verify the content is valid JSON
		var result map[string]interface{}
		err = json.Unmarshal([]byte(content), &result)
		require.NoError(t, err)

		assert.Equal(t, "info", result["level"])
		assert.Equal(t, "test message", result["message"])
		assert.Equal(t, "2023-01-01T00:00:00Z", result["time"])
	})

	t.Run("HandlesInvalidOuterJSON", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		invalidJSON := `{"invalid": json}`

		n, err := writer.Write([]byte(invalidJSON))

		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "unmarshal 1: invalid character 'j' looking for beginning of value", err.Error())
	})

	t.Run("HandlesInvalidInnerJSON", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		// Valid outer JSON but invalid inner JSON
		outerJSON := `{"message":"invalid json content"}`

		n, err := writer.Write([]byte(outerJSON))

		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "unmarshal 2: invalid character 'i' looking for beginning of value", err.Error())
	})

	t.Run("HandlesEmptyMessage", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		outerJSON := `{"message":""}`

		n, err := writer.Write([]byte(outerJSON))

		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "unmarshal 2: unexpected end of JSON input", err.Error())
	})

	t.Run("HandlesMultipleWrites", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		// First write
		innerJSON1 := `{"level":"info","message":"first message"}`
		outerJSON1 := `{"message":"` + strings.ReplaceAll(innerJSON1, `"`, `\"`) + `"}`

		n1, err1 := writer.Write([]byte(outerJSON1))
		require.NoError(t, err1)
		assert.Greater(t, n1, 0)

		// Second write
		innerJSON2 := `{"level":"error","message":"second message"}`
		outerJSON2 := `{"message":"` + strings.ReplaceAll(innerJSON2, `"`, `\"`) + `"}`

		n2, err2 := writer.Write([]byte(outerJSON2))
		require.NoError(t, err2)
		assert.Greater(t, n2, 0)

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Verify exact content - two JSON objects with newlines
		expectedContent := `{"level":"info","message":"first message"}
{"level":"error","message":"second message"}
`
		assert.Equal(t, expectedContent, content)
	})
}

func TestBufWriter_Content(t *testing.T) {
	t.Run("ReturnsEmptyStringInitially", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		content := writer.Content()
		assert.Equal(t, "", content)
	})

	t.Run("ReturnsWrittenContent", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		innerJSON := `{"level":"debug","message":"debug message"}`
		outerJSON := `{"message":"` + strings.ReplaceAll(innerJSON, `"`, `\"`) + `"}`

		_, err := writer.Write([]byte(outerJSON))
		require.NoError(t, err)

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Parse JSON to verify exact structure
		var result map[string]interface{}
		err = json.Unmarshal([]byte(content), &result)
		require.NoError(t, err)

		assert.Equal(t, "debug", result["level"])
		assert.Equal(t, "debug message", result["message"])
	})

	t.Run("ContentPersistsAcrossMultipleCalls", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		innerJSON := `{"level":"warn","message":"warning message"}`
		outerJSON := `{"message":"` + strings.ReplaceAll(innerJSON, `"`, `\"`) + `"}`

		_, err := writer.Write([]byte(outerJSON))
		require.NoError(t, err)

		content1 := writer.Content()
		content2 := writer.Content()

		assert.Equal(t, content1, content2)
		assert.NotEmpty(t, content1)
	})
}

func TestBufWriter_RealWorldUsage(t *testing.T) {
	t.Run("ProcessesWrappedJSONWithDifferentLogLevels", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		// Simulate different log levels in wrapped format
		testCases := []struct {
			level   string
			message string
		}{
			{"debug", "debug message"},
			{"info", "info message"},
			{"warn", "warning message"},
			{"error", "error message"},
		}

		for _, tc := range testCases {
			innerJSON := fmt.Sprintf(`{"level":"%s","message":"%s","time":"2023-01-01T00:00:00Z"}`, tc.level, tc.message)
			outerJSON := `{"message":"` + strings.ReplaceAll(innerJSON, `"`, `\"`) + `"}`

			_, err := writer.Write([]byte(outerJSON))
			require.NoError(t, err)
		}

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Verify exact content - all JSON objects with newlines
		expectedContent := `{"level":"debug","message":"debug message","time":"2023-01-01T00:00:00Z"}
{"level":"info","message":"info message","time":"2023-01-01T00:00:00Z"}
{"level":"warn","message":"warning message","time":"2023-01-01T00:00:00Z"}
{"level":"error","message":"error message","time":"2023-01-01T00:00:00Z"}
`
		assert.Equal(t, expectedContent, content)
	})

	t.Run("ProcessesWrappedJSONWithStructuredFields", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		innerJSON := `{"level":"info","message":"user logged in","user":"john","age":30,"active":true,"time":"2023-01-01T00:00:00Z"}`
		outerJSON := `{"message":"` + strings.ReplaceAll(innerJSON, `"`, `\"`) + `"}`

		_, err := writer.Write([]byte(outerJSON))
		require.NoError(t, err)

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Parse JSON to verify structure
		var result map[string]interface{}
		err = json.Unmarshal([]byte(content), &result)
		require.NoError(t, err)

		assert.Equal(t, "info", result["level"])
		assert.Equal(t, "user logged in", result["message"])
		assert.Equal(t, "john", result["user"])
		assert.Equal(t, float64(30), result["age"]) // JSON numbers are float64
		assert.Equal(t, true, result["active"])
		assert.Equal(t, "2023-01-01T00:00:00Z", result["time"])
	})

	t.Run("ProcessesWrappedJSONWithNestedObjects", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		innerJSON := `{"level":"info","message":"request processed","metadata":{"request_id":"12345","source":"api","tags":["important","user-action"]},"time":"2023-01-01T00:00:00Z"}`
		outerJSON := `{"message":"` + strings.ReplaceAll(innerJSON, `"`, `\"`) + `"}`

		_, err := writer.Write([]byte(outerJSON))
		require.NoError(t, err)

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Parse JSON to verify structure
		var result map[string]interface{}
		err = json.Unmarshal([]byte(content), &result)
		require.NoError(t, err)

		assert.Equal(t, "info", result["level"])
		assert.Equal(t, "request processed", result["message"])

		metadata, ok := result["metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "12345", metadata["request_id"])
		assert.Equal(t, "api", metadata["source"])

		tags, ok := metadata["tags"].([]interface{})
		require.True(t, ok)
		assert.Len(t, tags, 2)
		assert.Equal(t, "important", tags[0])
		assert.Equal(t, "user-action", tags[1])
	})

	t.Run("HandlesSpecialCharactersInMessages", func(t *testing.T) {
		writer := &testlogger.BufWriter{}

		// Use proper JSON marshaling to handle special characters correctly
		messageData := map[string]interface{}{
			"level":   "info",
			"message": `Message with "quotes", \backslashes\, and unicode: 🚀`,
			"time":    "2023-01-01T00:00:00Z",
		}

		innerJSONBytes, err := json.Marshal(messageData)
		require.NoError(t, err)

		outerData := map[string]string{
			"message": string(innerJSONBytes),
		}

		outerJSONBytes, err := json.Marshal(outerData)
		require.NoError(t, err)

		_, err = writer.Write(outerJSONBytes)
		require.NoError(t, err)

		content := writer.Content()
		assert.NotEmpty(t, content)

		// Parse the result to verify it's valid JSON
		var result map[string]interface{}
		err = json.Unmarshal([]byte(content), &result)
		require.NoError(t, err)

		// Should contain the special characters properly handled
		assert.Equal(t, "info", result["level"])
		assert.Equal(t, `Message with "quotes", \backslashes\, and unicode: 🚀`, result["message"])
		assert.Equal(t, "2023-01-01T00:00:00Z", result["time"])
	})
}
