# DriftMgr API Versioning Guide

## Overview

DriftMgr implements a complete API versioning strategy that ensures backward compatibility, smooth migration paths, and future extensibility. This guide covers all aspects of the versioning system.

## Versioning Strategy

### Semantic Versioning

DriftMgr follows [Semantic Versioning](https://semver.org/) for its API:
- **Major version** (X.y.z): Breaking changes that require client updates
- **Minor version** (x.Y.z): Backward-compatible feature additions
- **Patch version** (x.y.Z): Backward-compatible bug fixes

### Version Lifecycle

Each API version goes through a defined lifecycle:

1. **Alpha** - Early development, unstable API
2. **Beta** - Feature complete, API stabilizing
3. **Stable** - Production ready, fully supported
4. **Deprecated** - Marked for removal, migration encouraged
5. **Sunset** - No longer supported

## Supported Versions

| Version | Status | Release Date | Deprecation Date | Sunset Date | Description |
|---------|--------|--------------|------------------|-------------|-------------|
| 1.0.0 | Deprecated | 2024-01-01 | 2025-06-01 | 2026-01-01 | Initial stable release |
| 1.1.0 | Stable | 2024-06-01 | - | - | Enhanced API with plugin support |
| 2.0.0 | Beta | 2024-12-01 | - | - | Microservices architecture |
| 3.0.0 | Alpha | 2025-06-01 | - | - | AI-powered discovery |

## Version Negotiation Methods

### 1. Header-Based Versioning (Recommended)

The preferred method using the `API-Version` header:

```bash
curl -H "API-Version: 2.0.0" \
     -H "Accept: application/json" \
     https://api.driftmgr.io/api/discovery
```

**Advantages:**
- Clean URLs
- Easy to implement
- Works with all HTTP methods
- Can be cached appropriately

### 2. Content Negotiation

Using custom media types in the `Accept` header:

```bash
curl -H "Accept: application/vnd.driftmgr.v2+json" \
     https://api.driftmgr.io/api/discovery
```

**Media Type Format:**
- `application/vnd.driftmgr.v{major}+json` - Major version only
- `application/vnd.driftmgr.v{major}.{minor}+json` - Major.minor version
- `application/vnd.driftmgr.v{major}+stream` - Streaming endpoints (v2+)

### 3. Path-Based Versioning

Including version in the URL path:

```bash
curl https://api.driftmgr.io/v2/discovery
```

**Path Patterns:**
- `/v1/discovery` - Version 1.x endpoints
- `/v2/discovery` - Version 2.x endpoints
- `/api/discovery` - Version-agnostic (uses headers)

### 4. Query Parameter Versioning

Using `api_version` query parameter:

```bash
curl "https://api.driftmgr.io/api/discovery?api_version=2.0.0"
```

## Version-Specific Features

### Version 1.0.0 (Deprecated)

**Core Features:**
- Basic resource discovery
- Multi-cloud provider support (AWS, Azure, GCP)
- Account/subscription management
- Simple filtering capabilities

**Response Format:**
```json
{
  "api_version": {
    "major": 1,
    "minor": 0,
    "patch": 0,
    "status": "deprecated"
  },
  "data": {
    "resources": [...],
    "total": 150
  },
  "meta": {
    "deprecation": {
      "deprecated": true,
      "sunset_date": "2026-01-01T00:00:00Z",
      "migration_guide": "https://docs.driftmgr.io/migration/v2"
    }
  }
}
```

### Version 1.1.0 (Stable)

**New Features:**
- Plugin architecture support
- Dynamic provider loading
- Enhanced configuration system
- Improved error handling
- Advanced filtering and pagination

**Response Format:**
```json
{
  "api_version": {
    "major": 1,
    "minor": 1,
    "patch": 0,
    "status": "stable"
  },
  "data": {
    "resources": [...],
    "total": 150,
    "plugin_info": {
      "loaded_plugins": ["aws-v2", "azure-enhanced"]
    }
  }
}
```

### Version 2.0.0 (Beta)

**New Features:**
- Microservices architecture
- Streaming API endpoints
- Advanced caching strategies
- GraphQL support
- Real-time resource updates
- Resource relationship mapping
- Enhanced observability

**Response Format:**
```json
{
  "api_version": {
    "major": 2,
    "minor": 0,
    "patch": 0,
    "status": "beta"
  },
  "data": {
    "resources": [...],
    "pagination": {
      "total": 150,
      "page": 1,
      "has_next": true,
      "next_cursor": "eyJpZCI6InJlcy01MCJ9"
    },
    "relationships": [...],
    "_metadata": {
      "format_version": "2.0",
      "query_time": "120ms",
      "streaming_available": true
    }
  }
}
```

#### Streaming APIs (v2.0+)

Server-Sent Events for real-time updates:

```bash
curl -H "Accept: text/event-stream" \
     -H "API-Version: 2.0.0" \
     https://api.driftmgr.io/v2/resources/stream
```

**Stream Events:**
```
event: connected
data: {"timestamp": "2024-01-16T15:30:00Z", "stream_id": "stream-789"}

event: resource_update
data: {"event_type": "resource_discovered", "resource": {...}}

event: completed
data: {"total_resources": 150, "duration": "45s"}
```

## Error Handling

### Version Not Supported (410 Gone)

```json
{
  "error": "API version no longer supported",
  "version": "0.9.0",
  "sunset_date": "2024-01-01T00:00:00Z",
  "latest_version": "2.0.0",
  "migration_guide": "https://docs.driftmgr.io/migration/v2"
}
```

### Invalid Version Format (400 Bad Request)

```json
{
  "error": "Invalid API version",
  "message": "invalid version format: abc",
  "supported_versions": {
    "1.1.0": {
      "status": "stable",
      "description": "Enhanced API with plugin support"
    },
    "2.0.0": {
      "status": "beta",
      "description": "Microservices architecture"
    }
  }
}
```

### Compatibility Issues (400 Bad Request)

```json
{
  "error": "Request not compatible with API version",
  "version": "1.0.0",
  "issues": [
    "Field 'streaming_support' is not supported in API v1",
    "Field '_metadata' is not supported in API v1"
  ],
  "suggestions": [
    "Use API version 2.0.0 for streaming support",
    "Remove unsupported fields for v1.0.0 compatibility"
  ]
}
```

## Response Headers

### Standard Version Headers

All responses include version information in headers:

```http
API-Version: 2.0.0
API-Version-Status: beta
Content-Type: application/json
```

### Deprecation Warnings

Deprecated versions receive warning headers:

```http
Warning: 299 - "API version 1.0.0 is deprecated. Please migrate to 2.0.0"
Deprecation: true
Sunset: 2026-01-01T00:00:00Z
```

## Migration Guide

### From v1.0.0 to v1.1.0

**Breaking Changes:** None (backward compatible)

**New Features:**
- Plugin information in responses
- Enhanced metadata fields
- Improved error messages

**Migration Steps:**
1. Update client to handle new optional fields
2. Test with new plugin-specific responses
3. Optionally upgrade to use new filtering capabilities

### From v1.x to v2.0.0

**Breaking Changes:**
- Response structure changes
- New metadata format
- Pagination changes
- Field renames

**Migration Steps:**

1. **Update Response Parsing:**
   ```javascript
   // v1.x response
   const resources = response.data.resources;
   const total = response.data.total;
   
   // v2.0 response
   const resources = response.data.resources;
   const total = response.data.pagination.total;
   const metadata = response.data._metadata;
   ```

2. **Handle New Pagination:**
   ```javascript
   // v1.x pagination
   const page = response.data.page;
   
   // v2.0 cursor-based pagination
   const nextCursor = response.data.pagination.next_cursor;
   const hasNext = response.data.pagination.has_next;
   ```

3. **Update Field Names:**
   ```javascript
   // v1.x metadata
   const metadata = response.data.metadata;
   
   // v2.0 metadata
   const metadata = response.data._metadata;
   ```

4. **Use Streaming APIs (Optional):**
   ```javascript
   // v2.0 streaming
   const eventSource = new EventSource('/v2/resources/stream');
   eventSource.onmessage = (event) => {
     const data = JSON.parse(event.data);
     handleResourceUpdate(data);
   };
   ```

## Version Discovery

### Get Supported Versions

```bash
curl https://api.driftmgr.io/versions
```

**Response:**
```json
{
  "supported_versions": {
    "1.1.0": {
      "major": 1,
      "minor": 1,
      "patch": 0,
      "status": "stable",
      "release_date": "2024-06-01T00:00:00Z",
      "description": "Enhanced API with plugin support"
    },
    "2.0.0": {
      "major": 2,
      "minor": 0,
      "patch": 0,
      "status": "beta",
      "release_date": "2024-12-01T00:00:00Z",
      "description": "Microservices architecture"
    }
  },
  "default_version": "1.1.0",
  "latest_version": "2.0.0",
  "deprecation_policy": {
    "notice_period": "6 months",
    "support_period": "12 months after deprecation",
    "documentation": "https://docs.driftmgr.io/api/versioning"
  }
}
```

## Best Practices

### For API Clients

1. **Always specify version explicitly:**
   ```javascript
   const headers = {
     'API-Version': '2.0.0',
     'Accept': 'application/json'
   };
   ```

2. **Handle deprecation warnings:**
   ```javascript
   if (response.headers.deprecation === 'true') {
     console.warn('API version is deprecated:', response.headers.warning);
   }
   ```

3. **Implement graceful degradation:**
   ```javascript
   try {
     // Try v2.0 features
     const streamingData = await fetchStreamingData();
   } catch (error) {
     if (error.status === 400 && error.message.includes('not supported')) {
       // Fallback to v1.1 polling
       const pollingData = await fetchPollingData();
     }
   }
   ```

4. **Monitor sunset dates:**
   ```javascript
   const sunsetDate = response.headers.sunset;
   if (sunsetDate && new Date(sunsetDate) < new Date()) {
     // Plan migration urgently
   }
   ```

### For API Development

1. **Maintain backward compatibility within major versions**
2. **Use feature flags for gradual rollouts**
3. **Provide complete migration documentation**
4. **Implement automated compatibility testing**
5. **Monitor version usage analytics**

## Testing

### Version Compatibility Tests

```bash
# Test v1.0.0 compatibility
curl -H "API-Version: 1.0.0" \
     https://api.driftmgr.io/api/discovery

# Test v2.0.0 new features
curl -H "API-Version: 2.0.0" \
     https://api.driftmgr.io/api/discovery

# Test streaming (v2.0+ only)
curl -H "API-Version: 2.0.0" \
     -H "Accept: text/event-stream" \
     https://api.driftmgr.io/v2/resources/stream
```

### Automated Testing

```javascript
// Example Jest test for version compatibility
describe('API Versioning', () => {
  test('v1.1.0 should return stable response format', async () => {
    const response = await fetch('/api/discovery', {
      headers: { 'API-Version': '1.1.0' }
    });
    
    expect(response.headers.get('API-Version')).toBe('1.1.0');
    expect(response.headers.get('API-Version-Status')).toBe('stable');
    
    const data = await response.json();
    expect(data.api_version.major).toBe(1);
    expect(data.data.resources).toBeDefined();
  });
  
  test('v2.0.0 should include enhanced metadata', async () => {
    const response = await fetch('/api/discovery', {
      headers: { 'API-Version': '2.0.0' }
    });
    
    const data = await response.json();
    expect(data.data._metadata).toBeDefined();
    expect(data.data.pagination).toBeDefined();
  });
});
```

## Monitoring and Analytics

### Version Usage Metrics

- Track version distribution across client requests
- Monitor deprecation warning rates
- Alert on sunset version usage
- Measure migration success rates

### Performance Monitoring

- Compare response times across versions
- Monitor streaming connection stability (v2.0+)
- Track compatibility transformation overhead
- Measure cache hit rates by version

## Support and Documentation

- **API Documentation**: https://docs.driftmgr.io/api/
- **Migration Guides**: https://docs.driftmgr.io/migration/
- **Support Email**: api-support@driftmgr.io
- **GitHub Issues**: https://github.com/catherinevee/driftmgr/issues
- **Community Forum**: https://community.driftmgr.io/

## Conclusion

DriftMgr's API versioning system provides a robust foundation for evolving the API while maintaining compatibility and supporting smooth migrations. By following the guidelines in this document, developers can effectively integrate with and migrate between API versions as the platform evolves.