# Clean Architecture Migration Guide

## Quick Reference: Before vs After

### Current Structure (Layered Architecture)
```
internal/
├── device/
│   ├── handler/          → Depends on service
│   ├── service/          → Depends on repository (concrete)
│   ├── repository/       → Depends on database (concrete)
│   └── model/            → Mixed: domain + infrastructure
```

**Dependency Flow**: Handler → Service → Repository → Database
**Problem**: All dependencies point to concrete types

### Target Structure (Clean Architecture)
```
internal/
├── domain/device/        → No dependencies (pure Go)
│   ├── entity.go        → Pure domain entity
│   └── repository.go    → Interface definition
├── usecase/device/       → Depends on domain interfaces
│   └── service.go       → Business logic
├── delivery/http/       → Depends on usecase interfaces
│   └── handler/device.go
└── infrastructure/      → Implements domain interfaces
    └── database/postgres/device_repository.go
```

**Dependency Flow**: Infrastructure → Domain ← UseCase ← Delivery
**Solution**: Dependencies point inward to interfaces

## Step-by-Step Migration Example: Device Domain

### Step 1: Create Domain Layer

#### 1.1 Create Domain Entity (Pure, No GORM)
```go
// internal/domain/device/entity.go
package device

import (
    "time"
    "github.com/google/uuid"
)

type Device struct {
    ID                uuid.UUID
    HardwareUID       string
    DeviceName        *string
    Model             *string
    OwnerShipperID    *uuid.UUID
    CurrentShipmentID *uuid.UUID
    Status            DeviceStatus
    FirmwareVersion   *string
    BatteryLevel      *int
    TotalTrips        int
    LastSeenAt        *time.Time
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

type DeviceStatus string

const (
    StatusAvailable  DeviceStatus = "available"
    StatusInTransit  DeviceStatus = "in_transit"
    StatusMaintenance DeviceStatus = "maintenance"
    StatusRetired    DeviceStatus = "retired"
)
```

#### 1.2 Create Repository Interface
```go
// internal/domain/device/repository.go
package device

import (
    "context"
    "github.com/google/uuid"
)

type Repository interface {
    Create(ctx context.Context, device *Device) error
    GetByID(ctx context.Context, id uuid.UUID) (*Device, error)
    GetByHardwareUID(ctx context.Context, uid string) (*Device, error)
    Update(ctx context.Context, device *Device) error
    Delete(ctx context.Context, id uuid.UUID) error
    AssignOwner(ctx context.Context, deviceID, shipperID uuid.UUID) error
    UnassignOwner(ctx context.Context, deviceID uuid.UUID) error
    UpdateStatus(ctx context.Context, deviceID uuid.UUID, status DeviceStatus) error
    UpdateBattery(ctx context.Context, deviceID uuid.UUID, level int) error
    List(ctx context.Context, filter *Filter) ([]*Device, int64, error)
    GetStatistics(ctx context.Context) (*Statistics, error)
}

type Filter struct {
    Status         *DeviceStatus
    OwnerShipperID *uuid.UUID
    MinBattery     *int
    MaxBattery     *int
    IsOffline       *bool
    Search         string
    Page           int
    PageSize       int
    SortBy         string
    SortOrder       string
}
```

#### 1.3 Create Domain Errors
```go
// internal/domain/device/errors.go
package device

import "errors"

var (
    ErrDeviceNotFound      = errors.New("device not found")
    ErrDeviceAlreadyExists = errors.New("device already exists")
    ErrDeviceInUse         = errors.New("device is in use")
    ErrInvalidStatus       = errors.New("invalid device status")
)
```

### Step 2: Create Use Case Layer

```go
// internal/usecase/device/service.go
package device

import (
    "context"
    "logistics-quality-monitor/internal/domain/device"
    "logistics-quality-monitor/internal/domain/user"
    "github.com/google/uuid"
)

type Service struct {
    deviceRepo device.Repository
    userRepo   user.Repository
}

func NewService(deviceRepo device.Repository, userRepo user.Repository) *Service {
    return &Service{
        deviceRepo: deviceRepo,
        userRepo:   userRepo,
    }
}

func (s *Service) CreateDevice(ctx context.Context, req *CreateDeviceRequest) (*DeviceResponse, error) {
    // Validate request
    if req.HardwareUID == "" {
        return nil, device.ErrDeviceAlreadyExists
    }

    // Check if device exists
    existing, _ := s.deviceRepo.GetByHardwareUID(ctx, req.HardwareUID)
    if existing != nil {
        return nil, device.ErrDeviceAlreadyExists
    }

    // Validate owner if provided
    if req.OwnerShipperID != nil {
        owner, err := s.userRepo.GetByID(ctx, *req.OwnerShipperID)
        if err != nil {
            return nil, err
        }
        // Business rule: Only shippers can own devices
        if owner.Role != "shipper" {
            return nil, errors.New("only shippers can own devices")
        }
    }

    // Create domain entity
    newDevice := &device.Device{
        HardwareUID:     req.HardwareUID,
        DeviceName:      req.DeviceName,
        Model:           req.Model,
        OwnerShipperID:  req.OwnerShipperID,
        FirmwareVersion: req.FirmwareVersion,
        Status:          device.StatusAvailable,
    }

    // Save via repository interface
    if err := s.deviceRepo.Create(ctx, newDevice); err != nil {
        return nil, err
    }

    // Return response
    return toDeviceResponse(newDevice), nil
}

// Helper to convert domain entity to response DTO
func toDeviceResponse(d *device.Device) *DeviceResponse {
    return &DeviceResponse{
        ID:              d.ID,
        HardwareUID:     d.HardwareUID,
        DeviceName:      d.DeviceName,
        Model:           d.Model,
        OwnerShipperID:  d.OwnerShipperID,
        Status:          string(d.Status),
        // ... map other fields
    }
}
```

### Step 3: Create Infrastructure Implementation

#### 3.1 Create DB Model (with GORM tags)
```go
// internal/infrastructure/database/postgres/models/device_model.go
package models

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type DeviceModel struct {
    ID                uuid.UUID `gorm:"type:uuid;primary_key"`
    HardwareUID       string    `gorm:"type:varchar(255);uniqueIndex;not null"`
    DeviceName        *string   `gorm:"type:varchar(255)"`
    Model             *string   `gorm:"type:varchar(255)"`
    OwnerShipperID    *uuid.UUID `gorm:"type:uuid;index"`
    CurrentShipmentID *uuid.UUID `gorm:"type:uuid"`
    Status            string    `gorm:"type:varchar(50);not null;default:'available'"`
    FirmwareVersion   *string   `gorm:"type:varchar(100)"`
    BatteryLevel      *int      `gorm:"type:integer"`
    TotalTrips        int       `gorm:"type:integer;default:0"`
    LastSeenAt        *time.Time
    CreatedAt         time.Time
    UpdatedAt         time.Time

    OwnerShipper *UserModel `gorm:"foreignKey:OwnerShipperID;references:ID"`
}

func (DeviceModel) TableName() string {
    return "devices"
}
```

#### 3.2 Implement Repository Interface
```go
// internal/infrastructure/database/postgres/device_repository.go
package postgres

import (
    "context"
    "logistics-quality-monitor/internal/domain/device"
    "logistics-quality-monitor/internal/infrastructure/database/postgres/models"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

type DeviceRepository struct {
    db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) device.Repository {
    return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, d *device.Device) error {
    dbModel := toDBModel(d)
    
    if err := r.db.WithContext(ctx).Create(dbModel).Error; err != nil {
        return err
    }
    
    // Update domain entity with generated ID
    d.ID = dbModel.ID
    d.CreatedAt = dbModel.CreatedAt
    d.UpdatedAt = dbModel.UpdatedAt
    
    return nil
}

func (r *DeviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*device.Device, error) {
    var dbModel models.DeviceModel
    err := r.db.WithContext(ctx).
        Preload("OwnerShipper").
        Where("id = ?", id).
        First(&dbModel).Error
    
    if err == gorm.ErrRecordNotFound {
        return nil, device.ErrDeviceNotFound
    }
    if err != nil {
        return nil, err
    }
    
    return toDomainEntity(&dbModel), nil
}

// Helper functions to convert between domain and DB models
func toDBModel(d *device.Device) *models.DeviceModel {
    return &models.DeviceModel{
        ID:                d.ID,
        HardwareUID:       d.HardwareUID,
        DeviceName:        d.DeviceName,
        Model:             d.Model,
        OwnerShipperID:    d.OwnerShipperID,
        CurrentShipmentID: d.CurrentShipmentID,
        Status:            string(d.Status),
        FirmwareVersion:   d.FirmwareVersion,
        BatteryLevel:      d.BatteryLevel,
        TotalTrips:        d.TotalTrips,
        LastSeenAt:        d.LastSeenAt,
        CreatedAt:         d.CreatedAt,
        UpdatedAt:         d.UpdatedAt,
    }
}

func toDomainEntity(m *models.DeviceModel) *device.Device {
    status := device.DeviceStatus(m.Status)
    return &device.Device{
        ID:                m.ID,
        HardwareUID:       m.HardwareUID,
        DeviceName:        m.DeviceName,
        Model:             m.Model,
        OwnerShipperID:    m.OwnerShipperID,
        CurrentShipmentID: m.CurrentShipmentID,
        Status:            status,
        FirmwareVersion:   m.FirmwareVersion,
        BatteryLevel:      m.BatteryLevel,
        TotalTrips:        m.TotalTrips,
        LastSeenAt:        m.LastSeenAt,
        CreatedAt:         m.CreatedAt,
        UpdatedAt:         m.UpdatedAt,
    }
}
```

### Step 4: Update Delivery Layer

```go
// internal/delivery/http/handler/device_handler.go
package handler

import (
    "logistics-quality-monitor/internal/usecase/device"
    "github.com/gin-gonic/gin"
)

type DeviceHandler struct {
    service *device.Service
}

func NewDeviceHandler(service *device.Service) *DeviceHandler {
    return &DeviceHandler{service: service}
}

func (h *DeviceHandler) CreateDevice(c *gin.Context) {
    var req device.CreateDeviceRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        // Handle error
        return
    }
    
    resp, err := h.service.CreateDevice(c.Request.Context(), &req)
    if err != nil {
        // Handle error
        return
    }
    
    c.JSON(201, resp)
}
```

### Step 5: Wire Dependencies in main.go

```go
// cmd/main.go
package main

import (
    "logistics-quality-monitor/internal/delivery/http"
    "logistics-quality-monitor/internal/infrastructure/database/postgres"
    "logistics-quality-monitor/internal/usecase/device"
    "logistics-quality-monitor/internal/usecase/user"
)

func main() {
    // Initialize infrastructure
    db := postgres.NewDatabase(cfg)
    
    // Create repository implementations
    deviceRepo := postgres.NewDeviceRepository(db.DB)
    userRepo := postgres.NewUserRepository(db.DB)
    
    // Create use cases (depend on interfaces)
    deviceService := device.NewService(deviceRepo, userRepo)
    userService := user.NewService(userRepo, refreshRepo)
    
    // Create handlers (depend on use cases)
    deviceHandler := handler.NewDeviceHandler(deviceService)
    userHandler := handler.NewUserHandler(userService)
    
    // Setup routes
    router := http.SetupRoutes(deviceHandler, userHandler)
    
    // Start server
    // ...
}
```

## Migration Strategy

### Option 1: Gradual Migration (Recommended)
1. Start with one domain (e.g., `device`)
2. Create new structure alongside old
3. Migrate one use case at a time
4. Update tests
5. Remove old code once verified
6. Repeat for other domains

### Option 2: Big Bang Migration
1. Create entire new structure
2. Migrate all domains at once
3. Update all dependencies
4. Test thoroughly
5. Deploy

**Recommendation**: Use Option 1 for production systems

## Testing Strategy

### Unit Tests (Use Case Layer)
```go
// internal/usecase/device/service_test.go
func TestCreateDevice(t *testing.T) {
    // Mock repository interface
    mockRepo := &MockDeviceRepository{}
    mockUserRepo := &MockUserRepository{}
    
    service := NewService(mockRepo, mockUserRepo)
    
    // Test business logic without database
    // ...
}
```

### Integration Tests (Infrastructure Layer)
```go
// internal/infrastructure/database/postgres/device_repository_test.go
func TestDeviceRepository_Create(t *testing.T) {
    // Use test database
    db := setupTestDB(t)
    repo := NewDeviceRepository(db)
    
    // Test actual database operations
    // ...
}
```

## Checklist

- [ ] Create domain layer structure
- [ ] Extract pure domain entities
- [ ] Define repository interfaces
- [ ] Create use case services
- [ ] Implement repository in infrastructure
- [ ] Update handlers
- [ ] Update dependency injection
- [ ] Write unit tests with mocks
- [ ] Write integration tests
- [ ] Update documentation
- [ ] Remove old code

## Common Pitfalls to Avoid

1. ❌ Don't put GORM tags in domain entities
2. ❌ Don't import infrastructure packages in domain/use case
3. ❌ Don't skip interface definitions
4. ❌ Don't mix DTOs with domain entities
5. ❌ Don't forget to convert between layers

## Benefits After Migration

1. ✅ Easy to test (mock interfaces)
2. ✅ Easy to swap implementations
3. ✅ Clear separation of concerns
4. ✅ Business logic independent of frameworks
5. ✅ Better maintainability

