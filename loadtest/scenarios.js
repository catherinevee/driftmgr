import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const discoveryDuration = new Trend('discovery_duration');
const driftDetectionDuration = new Trend('drift_detection_duration');
const errorRate = new Rate('errors');
const successRate = new Rate('success');
const resourceCount = new Gauge('resource_count');
const concurrentUsers = new Counter('concurrent_users');

// Test configuration
export let options = {
    // Stages for ramping up load
    stages: [
        { duration: '2m', target: 50 },   // Ramp up to 50 users
        { duration: '5m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 200 },  // Spike to 200 users
        { duration: '3m', target: 200 },  // Stay at 200 users
        { duration: '2m', target: 0 },    // Ramp down to 0
    ],
    
    // Thresholds for pass/fail criteria
    thresholds: {
        'http_req_duration': ['p(95)<2000', 'p(99)<5000'], // 95% of requests < 2s, 99% < 5s
        'http_req_failed': ['rate<0.1'],                    // Error rate < 10%
        'errors': ['rate<0.1'],                             // Custom error rate < 10%
        'discovery_duration': ['p(95)<5000'],               // 95% of discoveries < 5s
        'drift_detection_duration': ['p(95)<3000'],         // 95% of drift detections < 3s
    },
    
    // Tags for better metrics organization
    tags: {
        environment: 'test',
        version: '1.0.0',
    },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

// Helper function to make authenticated requests
function makeRequest(method, endpoint, payload = null) {
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${API_KEY}`,
            'X-Request-ID': `k6-${Date.now()}-${Math.random()}`,
        },
        tags: { endpoint: endpoint },
        timeout: '30s',
    };
    
    let response;
    if (method === 'GET') {
        response = http.get(`${BASE_URL}${endpoint}`, params);
    } else if (method === 'POST') {
        response = http.post(`${BASE_URL}${endpoint}`, JSON.stringify(payload), params);
    } else if (method === 'PUT') {
        response = http.put(`${BASE_URL}${endpoint}`, JSON.stringify(payload), params);
    } else if (method === 'DELETE') {
        response = http.del(`${BASE_URL}${endpoint}`, null, params);
    }
    
    return response;
}

// Main test scenario
export default function() {
    concurrentUsers.add(1);
    
    // Scenario 1: Health Check
    group('Health Check', function() {
        const healthResponse = makeRequest('GET', '/health');
        
        const healthCheck = check(healthResponse, {
            'health check status is 200': (r) => r.status === 200,
            'health check response time < 500ms': (r) => r.timings.duration < 500,
            'system is healthy': (r) => {
                const body = JSON.parse(r.body);
                return body.status === 'healthy';
            },
        });
        
        if (!healthCheck) {
            errorRate.add(1);
        } else {
            successRate.add(1);
        }
    });
    
    sleep(1);
    
    // Scenario 2: Resource Discovery
    group('Resource Discovery', function() {
        const startTime = new Date();
        const discoveryResponse = makeRequest('POST', '/api/v1/discover', {
            providers: ['aws', 'azure', 'gcp'],
            regions: ['us-west-2', 'us-east-1'],
            includeDetails: true,
        });
        
        const duration = new Date() - startTime;
        discoveryDuration.add(duration);
        
        const discoveryCheck = check(discoveryResponse, {
            'discovery status is 200': (r) => r.status === 200,
            'discovery returns resources': (r) => {
                const body = JSON.parse(r.body);
                return body.resources && body.resources.length > 0;
            },
            'discovery time < 10s': (r) => r.timings.duration < 10000,
        });
        
        if (discoveryResponse.status === 200) {
            const body = JSON.parse(discoveryResponse.body);
            resourceCount.add(body.resources ? body.resources.length : 0);
            successRate.add(1);
        } else {
            errorRate.add(1);
        }
    });
    
    sleep(2);
    
    // Scenario 3: Drift Detection
    group('Drift Detection', function() {
        const startTime = new Date();
        const driftResponse = makeRequest('POST', '/api/v1/drift/detect', {
            provider: 'aws',
            stateFile: 'terraform.tfstate',
            compareWith: 'live',
        });
        
        const duration = new Date() - startTime;
        driftDetectionDuration.add(duration);
        
        const driftCheck = check(driftResponse, {
            'drift detection status is 200': (r) => r.status === 200,
            'drift detection completes': (r) => {
                const body = JSON.parse(r.body);
                return body.status === 'completed';
            },
            'drift detection time < 5s': (r) => r.timings.duration < 5000,
        });
        
        if (!driftCheck) {
            errorRate.add(1);
        } else {
            successRate.add(1);
        }
    });
    
    sleep(1);
    
    // Scenario 4: State Management
    group('State Management', function() {
        // List state files
        const listResponse = makeRequest('GET', '/api/v1/state/list');
        
        check(listResponse, {
            'state list status is 200': (r) => r.status === 200,
            'state list returns array': (r) => {
                const body = JSON.parse(r.body);
                return Array.isArray(body.states);
            },
        });
        
        // Get state details
        if (listResponse.status === 200) {
            const states = JSON.parse(listResponse.body).states;
            if (states && states.length > 0) {
                const stateId = states[0].id;
                const detailResponse = makeRequest('GET', `/api/v1/state/${stateId}`);
                
                check(detailResponse, {
                    'state detail status is 200': (r) => r.status === 200,
                    'state detail has resources': (r) => {
                        const body = JSON.parse(r.body);
                        return body.resources !== undefined;
                    },
                });
            }
        }
    });
    
    sleep(1);
    
    // Scenario 5: Concurrent Operations
    group('Concurrent Operations', function() {
        const batch = http.batch([
            ['GET', `${BASE_URL}/api/v1/providers`, null, { tags: { name: 'providers' } }],
            ['GET', `${BASE_URL}/api/v1/credentials`, null, { tags: { name: 'credentials' } }],
            ['GET', `${BASE_URL}/api/v1/metrics`, null, { tags: { name: 'metrics' } }],
        ]);
        
        check(batch[0], {
            'providers request successful': (r) => r.status === 200,
        });
        
        check(batch[1], {
            'credentials request successful': (r) => r.status === 200,
        });
        
        check(batch[2], {
            'metrics request successful': (r) => r.status === 200,
        });
    });
    
    sleep(2);
    
    // Scenario 6: Error Handling
    group('Error Handling', function() {
        // Test 404 handling
        const notFoundResponse = makeRequest('GET', '/api/v1/nonexistent');
        check(notFoundResponse, {
            '404 returns proper error': (r) => r.status === 404,
        });
        
        // Test invalid payload
        const invalidResponse = makeRequest('POST', '/api/v1/discover', {
            invalid: 'payload',
        });
        
        check(invalidResponse, {
            'invalid payload returns 400': (r) => r.status === 400,
        });
    });
    
    sleep(1);
}

// Setup function - runs once before the test
export function setup() {
    console.log('Starting DriftMgr load test...');
    
    // Verify system is ready
    const healthCheck = http.get(`${BASE_URL}/health`);
    if (healthCheck.status !== 200) {
        throw new Error('System health check failed');
    }
    
    return {
        startTime: new Date().toISOString(),
    };
}

// Teardown function - runs once after the test
export function teardown(data) {
    console.log(`Load test completed. Started at: ${data.startTime}`);
    
    // Optional: Send test results to monitoring system
    const summary = {
        startTime: data.startTime,
        endTime: new Date().toISOString(),
        environment: options.tags.environment,
    };
    
    // http.post(`${BASE_URL}/api/v1/test-results`, JSON.stringify(summary));
}

// Custom scenario for stress testing
export function stressTest() {
    // Aggressive discovery requests
    for (let i = 0; i < 10; i++) {
        makeRequest('POST', '/api/v1/discover', {
            providers: ['aws', 'azure', 'gcp', 'digitalocean'],
            parallel: true,
            timeout: 60000,
        });
    }
}

// Custom scenario for spike testing
export function spikeTest() {
    // Sudden burst of requests
    const batch = [];
    for (let i = 0; i < 100; i++) {
        batch.push(['GET', `${BASE_URL}/health`, null, { tags: { name: 'spike' } }]);
    }
    
    http.batch(batch);
}