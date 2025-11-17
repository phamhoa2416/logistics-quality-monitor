package ingestion

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Processor handles message processing with batching and concurrent workers
type Processor struct {
	repo        *Repository
	alertEngine *AlertEngine

	// Buffers for batching
	sensorBuffer  []SensorDataRecord
	locationCache map[string]*LocationDataMessage // deviceID -> latest location

	// Configuration
	batchSize    int
	batchTimeout time.Duration
	workerCount  int
	bufferSize   int

	// Channels
	sensorChan   chan *SensorDataMessage
	locationChan chan *LocationDataMessage

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex

	// Metrics
	metrics *MetricsTracker
}

// NewProcessor creates a new data processor
func NewProcessor(repo *Repository, alertEngine *AlertEngine, batchSize, workerCount, bufferSize int, batchTimeout time.Duration) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Processor{
		repo:          repo,
		alertEngine:   alertEngine,
		batchSize:     batchSize,
		batchTimeout:  batchTimeout,
		workerCount:   workerCount,
		bufferSize:    bufferSize,
		sensorBuffer:  make([]SensorDataRecord, 0, batchSize),
		locationCache: make(map[string]*LocationDataMessage),
		sensorChan:    make(chan *SensorDataMessage, bufferSize),
		locationChan:  make(chan *LocationDataMessage, bufferSize),
		ctx:           ctx,
		cancel:        cancel,
		metrics:       NewMetricsTracker(),
	}
}

// Start starts the processor workers
func (p *Processor) Start() {
	log.Printf("Starting processor with %d workers, batch size: %d, timeout: %v",
		p.workerCount, p.batchSize, p.batchTimeout)

	// Start sensor data workers
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.sensorWorker(i)
	}

	// Start location data worker
	p.wg.Add(1)
	go p.locationWorker()

	// Start batch flusher
	p.wg.Add(1)
	go p.batchFlusher()

	log.Println("Processor started")
}

// Stop stops the processor
func (p *Processor) Stop() {
	log.Println("Stopping processor...")

	p.cancel()

	// Close channels
	close(p.sensorChan)
	close(p.locationChan)

	// Wait for workers to finish
	p.wg.Wait()

	// Flush remaining data
	p.flushBatch()

	log.Println("Processor stopped")
}

// ProcessSensorData queues sensor data for processing
func (p *Processor) ProcessSensorData(msg *SensorDataMessage) {
	select {
	case p.sensorChan <- msg:
		p.metrics.Update(func(m *IngestMetrics) {
			m.MessagesReceived++
			m.BufferSize = len(p.sensorChan)
		})
	case <-p.ctx.Done():
		return
	default:
		log.Printf("Sensor buffer full, dropping message from device: %s", msg.DeviceID)
		p.metrics.Update(func(m *IngestMetrics) {
			m.MessagesFailed++
		})
	}
}

// ProcessLocationData queues location data for processing
func (p *Processor) ProcessLocationData(msg *LocationDataMessage) {
	if err := ValidateLocationData(msg); err != nil {
		log.Printf("Invalid location message: %v", err)
		p.metrics.Update(func(m *IngestMetrics) {
			m.MessagesFailed++
		})
		return
	}

	select {
	case p.locationChan <- msg:
		p.metrics.Update(func(m *IngestMetrics) {
			m.MessagesReceived++
		})
	case <-p.ctx.Done():
		return
	default:
		log.Printf("⚠️ Location buffer full, dropping message from device: %s", msg.DeviceID)
		p.metrics.Update(func(m *IngestMetrics) {
			m.MessagesFailed++
		})
	}
}

// sensorWorker processes sensor data messages
func (p *Processor) sensorWorker(id int) {
	defer p.wg.Done()

	log.Printf("Sensor worker %d started", id)

	for {
		select {
		case msg, ok := <-p.sensorChan:
			if !ok {
				return
			}

			start := time.Now()

			if err := p.processSensorMessage(msg); err != nil {
				log.Printf("Worker %d: Failed to process sensor message: %v", id, err)
				p.metrics.Update(func(m *IngestMetrics) {
					m.MessagesFailed++
				})
			} else {
				p.metrics.Update(func(m *IngestMetrics) {
					m.MessagesProcessed++
					m.LastProcessedAt = time.Now()

					// Update average processing time
					processingTime := time.Since(start)
					if m.AverageProcessingTime == 0 {
						m.AverageProcessingTime = processingTime
					} else {
						m.AverageProcessingTime = (m.AverageProcessingTime + processingTime) / 2
					}
				})
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// locationWorker processes location data messages
func (p *Processor) locationWorker() {
	defer p.wg.Done()

	log.Println("Location worker started")

	for {
		select {
		case msg, ok := <-p.locationChan:
			if !ok {
				return
			}

			// Cache latest location for device
			p.mu.Lock()
			p.locationCache[msg.DeviceID] = msg
			p.mu.Unlock()

			p.metrics.Update(func(m *IngestMetrics) {
				m.MessagesProcessed++
			})

		case <-p.ctx.Done():
			return
		}
	}
}

// processSensorMessage processes a single sensor message
func (p *Processor) processSensorMessage(msg *SensorDataMessage) error {
	// Validate message
	if err := ValidateSensorData(msg); err != nil {
		return err
	}

	// Parse device ID
	deviceID, err := uuid.Parse(msg.DeviceID)
	if err != nil {
		return err
	}

	// Get latest location for this device
	p.mu.Lock()
	location := p.locationCache[msg.DeviceID]
	p.mu.Unlock()

	// Create sensor data record
	record := SensorDataRecord{
		Time:           msg.Timestamp,
		DeviceID:       deviceID,
		Temperature:    msg.Temperature,
		Humidity:       msg.Humidity,
		LightLevel:     msg.LightLevel,
		TiltAngle:      msg.TiltAngle,
		ImpactG:        msg.ImpactG,
		BatteryLevel:   msg.BatteryLevel,
		SignalStrength: msg.SignalStrength,
	}

	// Add location if available
	if location != nil {
		record.Latitude = &location.Latitude
		record.Longitude = &location.Longitude
		record.Altitude = location.Altitude
		record.Speed = location.Speed
	}

	// Add to batch buffer
	p.mu.Lock()
	p.sensorBuffer = append(p.sensorBuffer, record)
	shouldFlush := len(p.sensorBuffer) >= p.batchSize
	p.mu.Unlock()

	// Flush if batch is full
	if shouldFlush {
		p.flushBatch()
	}

	// Check for violations and generate alerts (async)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		alerts, err := p.alertEngine.CheckViolations(ctx, &record)
		if err != nil {
			log.Printf("Failed to check violations: %v", err)
			return
		}

		if len(alerts) > 0 {
			if err := p.alertEngine.SaveAlerts(ctx, alerts); err != nil {
				log.Printf("Failed to save alerts: %v", err)
			} else {
				p.metrics.Update(func(m *IngestMetrics) {
					m.AlertsGenerated += int64(len(alerts))
				})
			}
		}
	}()

	// Update device last seen (async)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := p.repo.UpdateDeviceLastSeen(ctx, deviceID, msg.BatteryLevel); err != nil {
			log.Printf("Failed to update device last seen: %v", err)
		}
	}()

	return nil
}

// batchFlusher periodically flushes the batch
func (p *Processor) batchFlusher() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.flushBatch()
		case <-p.ctx.Done():
			return
		}
	}
}

// flushBatch writes buffered records to database
func (p *Processor) flushBatch() {
	p.mu.Lock()
	if len(p.sensorBuffer) == 0 {
		p.mu.Unlock()
		return
	}

	// Copy buffer and reset
	batch := make([]SensorDataRecord, len(p.sensorBuffer))
	copy(batch, p.sensorBuffer)
	p.sensorBuffer = p.sensorBuffer[:0]
	p.mu.Unlock()

	// Insert batch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	if err := p.repo.BatchInsertSensorData(ctx, batch); err != nil {
		log.Printf("❌ Failed to insert batch: %v", err)
		p.metrics.Update(func(m *IngestMetrics) {
			m.MessagesFailed += int64(len(batch))
		})
	} else {
		duration := time.Since(start)
		log.Printf("✅ Inserted batch of %d records in %v", len(batch), duration)
		p.metrics.Update(func(m *IngestMetrics) {
			m.RecordsInserted += int64(len(batch))
		})
	}
}

// GetMetrics returns current metrics
func (p *Processor) GetMetrics() IngestMetrics {
	return p.metrics.Snapshot()
}
