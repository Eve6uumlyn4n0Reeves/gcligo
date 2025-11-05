package features

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"gcli2api-go/internal/antitrunc"

	log "github.com/sirupsen/logrus"
)

// AntiTruncationConfig configuration for anti-truncation
type AntiTruncationConfig struct {
	MaxAttempts int
	Enabled     bool
}

// TruncationDetector detects if a response was truncated
type TruncationDetector struct {
	config     AntiTruncationConfig
	heuristics antitrunc.Config
}

// NewTruncationDetector creates a new truncation detector
func NewTruncationDetector(config AntiTruncationConfig) *TruncationDetector {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	heuristics := antitrunc.DefaultConfig()
	return &TruncationDetector{config: config, heuristics: heuristics}
}

// IsTruncated checks if a response appears to be truncated
func (td *TruncationDetector) IsTruncated(content string) bool {
	if !td.config.Enabled {
		return false
	}

	cleaned := antitrunc.CleanContinuationText(content)
	return td.heuristics.AppearsTruncated(cleaned)
}

// StreamBuffer buffers stream chunks and detects truncation
type StreamBuffer struct {
	chunks        []string
	buffer        *bytes.Buffer
	lastChunkTime time.Time
	timeout       time.Duration
}

// NewStreamBuffer creates a new stream buffer
func NewStreamBuffer(timeout time.Duration) *StreamBuffer {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &StreamBuffer{
		chunks:  make([]string, 0),
		buffer:  &bytes.Buffer{},
		timeout: timeout,
	}
}

// Add adds a chunk to the buffer
func (sb *StreamBuffer) Add(chunk string) {
	sb.chunks = append(sb.chunks, chunk)
	sb.buffer.WriteString(chunk)
	sb.lastChunkTime = time.Now()
}

// GetContent returns the buffered content
func (sb *StreamBuffer) GetContent() string {
	return sb.buffer.String()
}

// IsComplete checks if stream is complete (no new chunks in timeout period)
func (sb *StreamBuffer) IsComplete() bool {
	return time.Since(sb.lastChunkTime) > sb.timeout
}

// StreamHandler handles streaming with anti-truncation
type StreamHandler struct {
	detector *TruncationDetector
	config   AntiTruncationConfig
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(config AntiTruncationConfig) *StreamHandler {
	return &StreamHandler{
		detector: NewTruncationDetector(config),
		config:   config,
	}
}

// WrapStream wraps a stream reader with truncation detection
func (sh *StreamHandler) WrapStream(ctx context.Context, reader io.Reader, onTruncation func(context.Context) (io.Reader, error)) (io.Reader, error) {
	if !sh.config.Enabled {
		return reader, nil
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		buffer := NewStreamBuffer(3 * time.Second)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			buffer.Add(line + "\n")

			// Write to output
			if _, err := pw.Write([]byte(line + "\n")); err != nil {
				log.Warnf("Failed to write to pipe: %v", err)
				return
			}
		}

		if err := scanner.Err(); err != nil {
			log.Warnf("Scanner error: %v", err)
		}

		// Check for truncation
		content := buffer.GetContent()
		if sh.detector.IsTruncated(content) {
			log.Warn("Truncation detected, attempting continuation...")

			for attempt := 1; attempt <= sh.config.MaxAttempts; attempt++ {
				log.Infof("Continuation attempt %d/%d", attempt, sh.config.MaxAttempts)

				// Call onTruncation to get continuation
				contReader, err := onTruncation(ctx)
				if err != nil {
					log.Warnf("Continuation attempt %d failed: %v", attempt, err)
					break
				}

				// Read continuation
				contScanner := bufio.NewScanner(contReader)
				contScanner.Buffer(make([]byte, 64*1024), 1024*1024)

				contBuffer := NewStreamBuffer(3 * time.Second)
				for contScanner.Scan() {
					line := contScanner.Text()
					contBuffer.Add(line + "\n")

					if _, err := pw.Write([]byte(line + "\n")); err != nil {
						log.Warnf("Failed to write continuation: %v", err)
						return
					}
				}

				// Check if continuation is also truncated
				contContent := contBuffer.GetContent()
				if !sh.detector.IsTruncated(contContent) {
					log.Info("Continuation successful")
					break
				}

				if attempt >= sh.config.MaxAttempts {
					log.Warn("Max continuation attempts reached")
				}
			}
		}
	}()

	return pr, nil
}

// DetectAndHandle detects truncation in non-streaming response
func (sh *StreamHandler) DetectAndHandle(ctx context.Context, content string, onTruncation func(context.Context) (string, error)) (string, error) {
	if !sh.config.Enabled || !sh.detector.IsTruncated(content) {
		return content, nil
	}

	log.Warn("Truncation detected in response")

	var builder strings.Builder
	builder.Grow(len(content) + (len(content) / 2))
	builder.WriteString(content)
	for attempt := 1; attempt <= sh.config.MaxAttempts; attempt++ {
		log.Infof("Continuation attempt %d/%d", attempt, sh.config.MaxAttempts)

		contContent, err := onTruncation(ctx)
		if err != nil {
			return builder.String(), fmt.Errorf("continuation attempt %d failed: %w", attempt, err)
		}

		builder.WriteString(contContent)

		if !sh.detector.IsTruncated(contContent) {
			log.Info("Continuation successful")
			break
		}

		if attempt >= sh.config.MaxAttempts {
			log.Warn("Max continuation attempts reached")
		}
	}

	return builder.String(), nil
}
