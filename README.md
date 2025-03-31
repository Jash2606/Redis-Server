# Key-Value Cache Service

This is a high-performance in-memory Key-Value Cache service built with Go. The service provides a simple API for storing and retrieving key-value pairs with optimized performance for high traffic loads.

## Features

- In-memory key-value storage with fast access times
- Sharded cache design for improved concurrency
- CLOCK eviction algorithm to manage memory usage
- Memory monitoring to prevent out-of-memory errors
- Optimized for high throughput and low latency
- Thread-safe operations with fine-grained locking

## Setup Instructions

### Prerequisites

- Go 1.18 or higher
- Docker (for containerized deployment)

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/Jash2606/Redis-Server.git
   cd REDIS-SERVER
   ```

2. Build the application:
   ```bash
   go build -o key-value-cache
   ```

3. Run the application:
   ```bash
   ./key-value-cache
   ```

   The server will start on port 7171 by default.

### Docker Deployment

1. Build the Docker image:
   ```bash
   docker build -t key-value-cache .
   ```

2. Run the container:
   ```bash
   docker run -p 7171:7171 key-value-cache
   ```

## API Documentation

### PUT Operation

Inserts or updates a key-value pair in the cache.

- **URL**: `/put`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Request Body**:
  ```json
  {
    "key": "string (max 256 characters)",
    "value": "string (max 256 characters)"
  }
  ```
- **Success Response**:
  - **Code**: 200 OK
  - **Content**:
    ```json
    {
      "status": "OK",
      "message": "Key inserted/updated successfully."
    }
    ```
- **Error Responses**:
  - **Code**: 202 Accepted (for errors)
  - **Content Example**:
    ```json
    {
      "status": "ERROR",
      "message": "Invalid JSON"
    }
    ```

### GET Operation

Retrieves the value associated with a key.

- **URL**: `/get?key=`
- **Method**: `GET`
- **Success Response**:
  - **Code**: 200 OK
  - **Content**:
    ```json
    {
      "status": "OK",
      "key": "exampleKey",
      "value": "the corresponding value"
    }
    ```
- **Error Responses**:
  - **Code**: 202 Accepted
  - **Content Example**:
    ```json
    {
      "status": "ERROR",
      "message": "Key not found."
    }
    ```

## Testing

### Manual Testing with cURL

1. Test the PUT operation:
   ```bash
   curl -X POST http://localhost:7172/put \
     -H "Content-Type: application/json" \
     -d '{"key": "testKey", "value": "testValue"}'
   ```

2. Test the GET operation:
   ```bash
   curl -X GET "http://localhost:7172/get?key=testKey"
   ```

### Load Testing with Locust

1. Install Locust:
   ```bash
   pip install locust
   ```

2. Run the Locust test:
   ```bash
   locust -f locustfile.py --host=http://localhost:7172
   ```

3. Open the Locust web interface at http://localhost:8089 and start the test.

## Design Choices and Optimizations

1. **Sharded Cache**: The cache is divided into multiple shards to reduce lock contention and improve concurrent access.

2. **CLOCK Eviction Algorithm**: Implements an efficient CLOCK eviction policy that approximates LRU (Least Recently Used) without the overhead.

3. **Memory Management**: Continuously monitors memory usage and proactively evicts items when memory pressure increases.

4. **Object Pooling**: Uses sync.Pool to reuse request and response objects, reducing garbage collection pressure.

5. **Optimized HTTP Handlers**: Implements efficient request handling with context support and timeouts.

6. **Concurrent Request Handling**: Uses semaphores to limit concurrent requests and prevent server overload.

## Performance Considerations

- The service is optimized for high throughput and low latency.
- Default configuration supports up to 1000 concurrent requests.
- Memory usage is monitored and managed to prevent out-of-memory errors.
- The CLOCK eviction algorithm provides a good balance between performance and memory efficiency.

