package performance

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// CompressionManager provides data compression capabilities
type CompressionManager struct {
	config          *CompressionConfig
	compressionPool *CompressionPool
	stats           *CompressionStats
	mu              sync.RWMutex
}

// CompressionConfig defines compression behavior
type CompressionConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Algorithm        string `yaml:"algorithm"` // gzip, zlib, lz4
	CompressionLevel int    `yaml:"compression_level"`
	MinSizeThreshold int    `yaml:"min_size_threshold"`
	MaxSizeThreshold int    `yaml:"max_size_threshold"`
	CacheCompressed  bool   `yaml:"cache_compressed"`
	AutoOptimize     bool   `yaml:"auto_optimize"`
}

// CompressionStats tracks compression statistics
type CompressionStats struct {
	TotalCompressed   int64         `json:"total_compressed"`
	TotalUncompressed int64         `json:"total_uncompressed"`
	CompressionRatio  float64       `json:"compression_ratio"`
	AverageTime       time.Duration `json:"average_time"`
	HitCount          int64         `json:"hit_count"`
	MissCount         int64         `json:"miss_count"`
	LastReset         time.Time     `json:"last_reset"`
}

// CompressedData represents compressed data with metadata
type CompressedData struct {
	Data            []byte        `json:"data"`
	OriginalSize    int           `json:"original_size"`
	CompressedSize  int           `json:"compressed_size"`
	Algorithm       string        `json:"algorithm"`
	CompressionTime time.Duration `json:"compression_time"`
	Timestamp       time.Time     `json:"timestamp"`
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager(config *CompressionConfig) *CompressionManager {
	if config == nil {
		config = &CompressionConfig{
			Enabled:          true,
			Algorithm:        "gzip",
			CompressionLevel: 6,
			MinSizeThreshold: 1024,             // 1KB
			MaxSizeThreshold: 10 * 1024 * 1024, // 10MB
			CacheCompressed:  true,
			AutoOptimize:     true,
		}
	}

	return &CompressionManager{
		config:          config,
		compressionPool: NewCompressionPool(config),
		stats: &CompressionStats{
			LastReset: time.Now(),
		},
	}
}

// Compress compresses data
func (cm *CompressionManager) Compress(data []byte) (*CompressedData, error) {
	if !cm.config.Enabled {
		return &CompressedData{
			Data:           data,
			OriginalSize:   len(data),
			CompressedSize: len(data),
			Algorithm:      "none",
			Timestamp:      time.Now(),
		}, nil
	}

	// Check size thresholds
	if len(data) < cm.config.MinSizeThreshold {
		return &CompressedData{
			Data:           data,
			OriginalSize:   len(data),
			CompressedSize: len(data),
			Algorithm:      "none",
			Timestamp:      time.Now(),
		}, nil
	}

	if len(data) > cm.config.MaxSizeThreshold {
		return nil, fmt.Errorf("data size %d exceeds maximum threshold %d", len(data), cm.config.MaxSizeThreshold)
	}

	start := time.Now()

	var compressed []byte
	var err error

	switch cm.config.Algorithm {
	case "gzip":
		compressed, err = cm.compressGzip(data)
	case "zlib":
		compressed, err = cm.compressZlib(data)
	default:
		compressed, err = cm.compressGzip(data)
	}

	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	compressionTime := time.Since(start)

	// Update statistics
	cm.updateStats(len(data), len(compressed), compressionTime)

	return &CompressedData{
		Data:            compressed,
		OriginalSize:    len(data),
		CompressedSize:  len(compressed),
		Algorithm:       cm.config.Algorithm,
		CompressionTime: compressionTime,
		Timestamp:       time.Now(),
	}, nil
}

// Decompress decompresses data
func (cm *CompressionManager) Decompress(compressedData *CompressedData) ([]byte, error) {
	if compressedData.Algorithm == "none" {
		return compressedData.Data, nil
	}

	start := time.Now()

	var decompressed []byte
	var err error

	switch compressedData.Algorithm {
	case "gzip":
		decompressed, err = cm.decompressGzip(compressedData.Data)
	case "zlib":
		decompressed, err = cm.decompressZlib(compressedData.Data)
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", compressedData.Algorithm)
	}

	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	_ = time.Since(start) // decompressionTime not used yet

	// Verify size
	if len(decompressed) != compressedData.OriginalSize {
		return nil, fmt.Errorf("decompressed size %d doesn't match original size %d",
			len(decompressed), compressedData.OriginalSize)
	}

	return decompressed, nil
}

// CompressJSON compresses JSON data
func (cm *CompressionManager) CompressJSON(data interface{}) (*CompressedData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("JSON marshaling failed: %w", err)
	}

	return cm.Compress(jsonData)
}

// DecompressJSON decompresses JSON data
func (cm *CompressionManager) DecompressJSON(compressedData *CompressedData, target interface{}) error {
	decompressed, err := cm.Decompress(compressedData)
	if err != nil {
		return err
	}

	err = json.Unmarshal(decompressed, target)
	if err != nil {
		return fmt.Errorf("JSON unmarshaling failed: %w", err)
	}

	return nil
}

// OptimizeCompression optimizes compression settings
func (cm *CompressionManager) OptimizeCompression() error {
	if !cm.config.AutoOptimize {
		return nil
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Analyze current performance
	ratio := cm.stats.CompressionRatio
	avgTime := cm.stats.AverageTime

	// Adjust compression level based on performance
	if ratio < 0.5 && avgTime < 100*time.Millisecond {
		// Low compression ratio, fast compression - increase level
		if cm.config.CompressionLevel < 9 {
			cm.config.CompressionLevel++
			fmt.Printf("Increased compression level to %d\n", cm.config.CompressionLevel)
		}
	} else if ratio > 0.8 && avgTime > 500*time.Millisecond {
		// High compression ratio, slow compression - decrease level
		if cm.config.CompressionLevel > 1 {
			cm.config.CompressionLevel--
			fmt.Printf("Decreased compression level to %d\n", cm.config.CompressionLevel)
		}
	}

	return nil
}

// GetStatistics returns compression statistics
func (cm *CompressionManager) GetStatistics() *CompressionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := *cm.stats

	// Calculate compression ratio
	if stats.TotalUncompressed > 0 {
		stats.CompressionRatio = float64(stats.TotalCompressed) / float64(stats.TotalUncompressed)
	}

	return &stats
}

// ResetStatistics resets compression statistics
func (cm *CompressionManager) ResetStatistics() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats = &CompressionStats{
		LastReset: time.Now(),
	}
}

// compressGzip compresses data using gzip
func (cm *CompressionManager) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	writer, err := gzip.NewWriterLevel(&buf, cm.config.CompressionLevel)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressGzip decompresses gzip data
func (cm *CompressionManager) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// compressZlib compresses data using zlib
func (cm *CompressionManager) compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	writer, err := zlib.NewWriterLevel(&buf, cm.config.CompressionLevel)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressZlib decompresses zlib data
func (cm *CompressionManager) decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// updateStats updates compression statistics
func (cm *CompressionManager) updateStats(originalSize, compressedSize int, compressionTime time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats.TotalUncompressed += int64(originalSize)
	cm.stats.TotalCompressed += int64(compressedSize)

	// Update average time
	totalCount := cm.stats.HitCount + cm.stats.MissCount
	if totalCount > 0 {
		totalTime := cm.stats.AverageTime * time.Duration(totalCount)
		cm.stats.AverageTime = (totalTime + compressionTime) / time.Duration(totalCount+1)
	} else {
		cm.stats.AverageTime = compressionTime
	}

	cm.stats.HitCount++
}

// CompressionPool manages compression resources
type CompressionPool struct {
	config *CompressionConfig
	mu     sync.Mutex
}

// NewCompressionPool creates a new compression pool
func NewCompressionPool(config *CompressionConfig) *CompressionPool {
	return &CompressionPool{
		config: config,
	}
}

// GetCompressionLevel returns the current compression level
func (cp *CompressionPool) GetCompressionLevel() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.config.CompressionLevel
}

// SetCompressionLevel sets the compression level
func (cp *CompressionPool) SetCompressionLevel(level int) error {
	if level < 1 || level > 9 {
		return fmt.Errorf("compression level must be between 1 and 9")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.config.CompressionLevel = level
	return nil
}

// CompressionBenchmark benchmarks compression performance
type CompressionBenchmark struct {
	Algorithm         string        `json:"algorithm"`
	Level             int           `json:"level"`
	OriginalSize      int           `json:"original_size"`
	CompressedSize    int           `json:"compressed_size"`
	CompressionTime   time.Duration `json:"compression_time"`
	DecompressionTime time.Duration `json:"decompression_time"`
	Ratio             float64       `json:"ratio"`
	Speed             float64       `json:"speed_mbps"`
}

// BenchmarkCompression benchmarks compression algorithms
func (cm *CompressionManager) BenchmarkCompression(data []byte) ([]*CompressionBenchmark, error) {
	algorithms := []string{"gzip", "zlib"}
	levels := []int{1, 6, 9}

	var benchmarks []*CompressionBenchmark

	for _, algorithm := range algorithms {
		for _, level := range levels {
			// Temporarily set compression level
			originalLevel := cm.config.CompressionLevel
			cm.config.CompressionLevel = level

			// Benchmark compression
			start := time.Now()
			compressed, err := cm.Compress(data)
			compressionTime := time.Since(start)

			if err != nil {
				cm.config.CompressionLevel = originalLevel
				continue
			}

			// Benchmark decompression
			start = time.Now()
			_, err = cm.Decompress(compressed)
			decompressionTime := time.Since(start)

			if err != nil {
				cm.config.CompressionLevel = originalLevel
				continue
			}

			// Calculate metrics
			ratio := float64(compressed.CompressedSize) / float64(compressed.OriginalSize)
			speed := float64(compressed.OriginalSize) / compressionTime.Seconds() / 1024 / 1024 // MB/s

			benchmark := &CompressionBenchmark{
				Algorithm:         algorithm,
				Level:             level,
				OriginalSize:      compressed.OriginalSize,
				CompressedSize:    compressed.CompressedSize,
				CompressionTime:   compressionTime,
				DecompressionTime: decompressionTime,
				Ratio:             ratio,
				Speed:             speed,
			}

			benchmarks = append(benchmarks, benchmark)

			// Restore original level
			cm.config.CompressionLevel = originalLevel
		}
	}

	return benchmarks, nil
}

// GetOptimalSettings returns optimal compression settings based on data characteristics
func (cm *CompressionManager) GetOptimalSettings(dataSize int, dataType string) (string, int) {
	// Simple heuristic for optimal settings
	// In a real implementation, this would use machine learning or historical data

	if dataSize < 1024 {
		return "gzip", 1 // Fast compression for small data
	} else if dataSize < 1024*1024 {
		return "gzip", 6 // Balanced for medium data
	} else {
		return "gzip", 9 // Maximum compression for large data
	}
}
