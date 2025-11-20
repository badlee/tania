# 🧪 Tests Documentation

## Structure

```
tests/
├── main.go           # Main Go tests
├── client.js         # JavaScript client tests
├── data.sql          # Test fixtures
└── coverage/         # Coverage reports
```

## Running Tests

### All tests
```bash
./test.sh
# or
make test-all
```

### Go tests only
```bash
make test-go
# or
go test -v ./...
```

### JavaScript tests only
```bash
make test-js
# or
bun test --timeout 10000
```

### With coverage
```bash
make coverage
```

### Benchmarks
```bash
make benchmark
```

### Specific tests
```bash
go test -v -run TestLocationManager
go test -v -run TestFollowUser
go test -v -run TestCreateRoom
```

## Test Categories

### Unit Tests
- PubSub functionality
- Location calculations (Haversine, point-in-polygon)
- Follow/Follower logic
- Room management
- User channels (SSE, WebRTC)

### Integration Tests
- Full user workflows
- Marketplace transactions
- Follow approval flows
- Room creation and joining

### Benchmark Tests
- Haversine distance calculation
- Finding nearby users (1000+ users)
- PubSub publish performance

## Coverage Goals

- **Go Code**: > 80% coverage
- **JavaScript**: > 70% coverage

## Continuous Integration

Tests run automatically on:
- Push to main/develop
- Pull requests

See `.github/workflows/test.yml`

## Writing Tests

### Go Test Example
```go
func TestMyFeature(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()

    // Test code
    assert.Equal(t, expected, actual)
}
```

### JavaScript Test Example
```javascript
describe('MyFeature', () => {
  test('should work correctly', async () => {
    const result = await myFunction();
    expect(result).toBe(expected);
  });
});
```

## Test Data

Use `tests/data.sql` for consistent test fixtures.

## Mock Data

Tests use mock PocketBase instances and don't require a running server.
