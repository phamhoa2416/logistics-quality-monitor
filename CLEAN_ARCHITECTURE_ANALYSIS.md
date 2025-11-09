# Clean Architecture Analysis & Recommendations

## Executive Summary

Your project currently follows a **layered architecture** pattern but does **not** fully adhere to **Clean Architecture** principles. The main issues are:

1. ❌ **No Dependency Inversion**: Services depend on concrete repositories instead of interfaces
2. ❌ **No Domain Layer Separation**: Business entities are mixed with infrastructure concerns (GORM tags)
3. ❌ **Tight Coupling**: Infrastructure details leak into business logic
4. ❌ **Missing Interface Definitions**: Repository interfaces should be in domain/use case layer
5. ⚠️ **Infrastructure in Internal**: Database and config should be clearly separated

## Current Structure Analysis

### Current Layout
```
internal/
├── config/              # Infrastructure concern
├── database/            # Infrastructure concern  
├── device/
│   ├── handler/         # Presentation layer
│   ├── model/           # Mixed: entities + DTOs + GORM tags
│   ├── repository/      # Infrastructure (concrete implementation)
│   ├── service/         # Business logic (but depends on concrete repos)
│   └── validator/       # Business logic
├── user/
│   ├── handler/         # Presentation layer
│   ├── model/           # Mixed: entities + DTOs + GORM tags
│   ├── repository/      # Infrastructure (concrete implementation)
│   └── service/         # Business logic
├── middleware/          # Presentation layer concern
├── logger/              # Infrastructure concern
└── routes/              # Presentation layer
```

### Issues Identified

#### 1. **No Dependency Inversion Principle (DIP)**
- **Problem**: Services depend on concrete repository types
  ```go
  // Current (BAD)
  type DeviceService struct {
      repo *repository.DeviceRepository  // Concrete type
  }
  ```
- **Should be**: Services depend on interfaces defined in domain/use case layer
  ```go
  // Clean Architecture (GOOD)
  type DeviceService struct {
      repo domain.DeviceRepository  // Interface
  }
  ```

#### 2. **Domain Entities Mixed with Infrastructure**
- **Problem**: Models contain GORM tags and database-specific concerns
  ```go
  // Current (BAD)
  type Device struct {
      ID uuid.UUID `json:"id" gorm:"primaryKey"`  // Infrastructure concern
      OwnerShipper *user.User `gorm:"foreignKey:OwnerShipperID"`  // GORM specific
  }
  ```
- **Should be**: Pure domain entities without infrastructure tags

#### 3. **Tight Coupling**
- Services import infrastructure packages directly
- Cannot easily swap implementations (e.g., PostgreSQL → MongoDB)
- Hard to test without database

#### 4. **No Clear Domain Layer**
- Business rules are scattered
- No clear separation between domain logic and application logic

## Recommended Clean Architecture Structure

### Proposed Structure
```
cmd/
  main.go                    # Application entry point

internal/
  domain/                    # Domain Layer (Innermost - No dependencies)
    user/
      entity.go              # Pure domain entity (no GORM tags)
      repository.go          # Repository interface
      errors.go              # Domain-specific errors
    device/
      entity.go              # Pure domain entity
      repository.go          # Repository interface
      errors.go              # Domain-specific errors
  
  usecase/                    # Application/Use Case Layer
    user/
      service.go             # Implements business logic
      interfaces.go          # Use case interfaces (optional)
    device/
      service.go             # Implements business logic
      interfaces.go          # Use case interfaces (optional)
  
  delivery/                  # Presentation/Interface Adapters Layer
    http/
      handler/
        user_handler.go      # HTTP handlers
        device_handler.go
      dto/                   # Data Transfer Objects
        user_dto.go          # Request/Response DTOs
        device_dto.go
      router.go              # HTTP routing
    middleware/              # HTTP middleware
      auth.go
      cors.go
      ...
  
  infrastructure/            # Infrastructure/External Layer
    database/
      postgres/
        connection.go        # Database connection
        user_repository.go   # Implements domain.UserRepository
        device_repository.go # Implements domain.DeviceRepository
    config/
      config.go
    logger/
      logger.go
    cache/                   # Future: Redis, etc.
    messaging/               # Future: RabbitMQ, etc.

pkg/                         # Shared utilities (no business logic)
  errors/
  utils/
```

## Key Principles Applied

### 1. **Dependency Rule**
- **Domain** → No dependencies (pure Go types)
- **Use Case** → Depends only on Domain interfaces
- **Delivery** → Depends on Use Case interfaces
- **Infrastructure** → Implements Domain interfaces

### 2. **Dependency Inversion**
- Inner layers define interfaces
- Outer layers implement interfaces
- Dependencies point inward

### 3. **Separation of Concerns**
- **Domain**: Business entities and rules
- **Use Case**: Application-specific business logic
- **Delivery**: HTTP, gRPC, CLI adapters
- **Infrastructure**: Database, external services

## Migration Plan

### Phase 1: Create Domain Layer
1. Extract pure domain entities (remove GORM tags)
2. Define repository interfaces in domain layer
3. Move domain-specific errors to domain layer

### Phase 2: Refactor Use Case Layer
1. Create use case services that depend on domain interfaces
2. Move business logic from current services to use cases
3. Keep application-specific logic separate from domain logic

### Phase 3: Refactor Infrastructure
1. Move database implementations to infrastructure layer
2. Implement domain repository interfaces
3. Move config, logger to infrastructure

### Phase 4: Refactor Delivery Layer
1. Move handlers to delivery/http/handler
2. Create DTOs for request/response
3. Move middleware to delivery/http/middleware
4. Update routing

### Phase 5: Update Dependencies
1. Update main.go to wire dependencies
2. Use dependency injection
3. Update all imports

## Code Examples

### Domain Entity (Pure)
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
    StatusInTransit DeviceStatus = "in_transit"
    StatusMaintenance DeviceStatus = "maintenance"
    StatusRetired   DeviceStatus = "retired"
)
```

### Domain Repository Interface
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
    List(ctx context.Context, filter *Filter) ([]*Device, int64, error)
    // ... other methods
}
```

### Use Case Service
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
    // Business logic here
    // Uses domain.Repository interface, not concrete type
}
```

### Infrastructure Implementation
```go
// internal/infrastructure/database/postgres/device_repository.go
package postgres

import (
    "context"
    "logistics-quality-monitor/internal/domain/device"
    "logistics-quality-monitor/internal/infrastructure/database"
    "github.com/google/uuid"
)

type DeviceRepository struct {
    db *database.DB
}

func NewDeviceRepository(db *database.DB) device.Repository {
    return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, d *device.Device) error {
    // Convert domain entity to database model
    dbModel := toDBModel(d)
    // Use GORM here
    return r.db.DB.WithContext(ctx).Create(dbModel).Error
}

// Helper to convert domain entity to DB model
func toDBModel(d *device.Device) *DeviceModel {
    return &DeviceModel{
        ID:          d.ID,
        HardwareUID: d.HardwareUID,
        // ... map fields
    }
}
```

## Benefits of This Structure

1. ✅ **Testability**: Easy to mock interfaces for unit testing
2. ✅ **Flexibility**: Swap implementations (PostgreSQL → MongoDB) without changing business logic
3. ✅ **Maintainability**: Clear separation of concerns
4. ✅ **Scalability**: Easy to add new features without affecting existing code
5. ✅ **Independence**: Business logic independent of frameworks, databases, UI

## Go-Specific Best Practices

1. **Package Organization**: Follow Go conventions
   - One package per domain aggregate
   - Interfaces in domain layer
   - Implementations in infrastructure

2. **Dependency Injection**: Use constructor injection
   ```go
   func NewService(repo Repository) *Service {
       return &Service{repo: repo}
   }
   ```

3. **Error Handling**: Domain-specific errors in domain layer
   ```go
   // internal/domain/device/errors.go
   var ErrDeviceNotFound = errors.New("device not found")
   ```

4. **Context Propagation**: Always pass context through layers

## Next Steps

1. Review this analysis
2. Decide on migration approach (gradual vs. big bang)
3. Start with one domain (e.g., `device`) as a proof of concept
4. Gradually migrate other domains
5. Update tests to use interfaces

## Additional Recommendations

1. **Add Interface Segregation**: Split large repository interfaces into smaller ones
2. **Add Value Objects**: Extract complex types (e.g., Email, PhoneNumber)
3. **Add Domain Events**: For cross-aggregate communication
4. **Add Use Case Interfaces**: If you need to swap implementations
5. **Consider CQRS**: Separate read/write models if needed

