package main

// longTextParagraphs contains predefined long text content for generating queries.
// Each paragraph is approximately 500-1000 tokens.
var longTextParagraphs = []string{
	`## Software Engineering Best Practices

Software engineering is the application of engineering principles to software development, maintenance, and testing. Here are some key best practices:

### 1. Code Quality
- **Readability**: Code should be as easy to read as prose. Use meaningful variable names, function names, and comments.
- **Modularity**: Break code into small, reusable functions and classes.
- **DRY Principle**: Don't Repeat Yourself - avoid code duplication.
- **SOLID Principles**: Single responsibility, open-closed, Liskov substitution, interface segregation, dependency inversion.

### 2. Version Control
Git is currently the most popular version control system. Key practices include:
- Commit frequently, with each commit containing one logical change
- Use clear commit messages
- Use branches for feature development and bug fixes
- Code review to ensure code quality

### 3. Testing
- **Unit Testing**: Test individual functions and methods
- **Integration Testing**: Test interactions between components
- **End-to-End Testing**: Test complete user flows
- Test coverage should remain high (typically >80%)

### 4. CI/CD
Automated build, test, and deployment pipelines:
- Automatically run tests on each code commit
- Automatically generate and deploy build artifacts
- Provide fast feedback to the development team
`,

	`## Python Asynchronous Programming

Python's asyncio library provides powerful async programming support.

### Defining Async Functions
Using async/await syntax:

import asyncio

async def fetch_data(url: str) -> dict:
    # Simulate async IO operation
    await asyncio.sleep(1)
    return {"url": url, "data": "sample"}

async def main():
    # Execute multiple tasks concurrently
    tasks = [
        fetch_data("https://api.example.com/1"),
        fetch_data("https://api.example.com/2"),
        fetch_data("https://api.example.com/3"),
    ]
    results = await asyncio.gather(*tasks)
    print(results)

asyncio.run(main())

### Event Loop
asyncio is based on the event loop model:
- The event loop is responsible for scheduling and executing coroutines
- When a coroutine waits for IO, the event loop switches to other coroutines
- Implements single-threaded concurrent execution

### Common Patterns
1. **asyncio.gather()**: Execute multiple coroutines concurrently
2. **asyncio.wait()**: More flexible wait control
3. **asyncio.create_task()**: Create background tasks
4. **asyncio.Queue**: Async-safe queue
5. **asyncio.Lock/Semaphore**: Async synchronization primitives

Async programming can significantly improve performance for IO-intensive applications.
`,

	`## Distributed System Design Principles

Designing distributed systems requires considering several key factors:

### CAP Theorem
In distributed systems, consistency, availability, and partition tolerance cannot all be achieved simultaneously. Typically, you must choose between CP and AP.

### Consistency Models
1. **Strong Consistency**: All nodes see the same data simultaneously
2. **Eventual Consistency**: The system guarantees that if no new updates are made, eventually all accesses will return the last updated value
3. **Causal Consistency**: Causally related operations are visible in the same order to all processes

### Data Sharding Strategies
- **Horizontal Sharding**: Split data by rows
- **Vertical Sharding**: Split data by columns
- **Hash Sharding**: Use hash function to distribute data
- **Range Sharding**: Distribute data by key ranges

### Replication Strategies
1. **Master-Slave Replication**: Writes on master, reads from slaves
2. **Multi-Master Replication**: Multiple nodes can accept writes
3. **Leaderless Replication**: No explicit master node

### Fault Tolerance Mechanisms
- **Heartbeat Detection**: Monitor node health
- **Failover**: Automatically switch to backup nodes
- **Data Redundancy**: Store multiple copies of critical data
`,

	`## Deep Learning Fundamentals

Deep learning is a branch of machine learning that uses multi-layer neural networks.

### Neural Network Layer Types
1. **Dense Layer (Fully Connected)**: Each input connects to each output
2. **Convolutional Layer (Conv2D)**: Extract spatial features
3. **Recurrent Layer (LSTM/GRU)**: Process sequence data
4. **Transformer Layer**: Self-attention mechanism

### Activation Functions
- **ReLU**: max(0, x) - Most commonly used
- **Sigmoid**: 1/(1+e^(-x)) - Output 0-1
- **Tanh**: (e^x - e^(-x))/(e^x + e^(-x)) - Output -1 to 1
- **Softmax**: Multi-class normalization

### Loss Functions
- **MSE**: Regression tasks
- **Cross Entropy**: Classification tasks
- **Binary Cross Entropy**: Binary classification

### Optimization Algorithms
1. **SGD**: Stochastic Gradient Descent
2. **Adam**: Adaptive learning rate
3. **RMSprop**: Similar to Adam

### Regularization Techniques
- **Dropout**: Randomly drop neurons
- **Batch Normalization**: Normalize layer inputs
- **L1/L2 Regularization**: Weight penalty

Deep learning has achieved breakthrough progress in image recognition, natural language processing, and speech recognition.
`,

	`## Database Transaction Processing

Transactions are a fundamental concept in database management systems, ensuring data consistency.

### ACID Properties
- **Atomicity**: A transaction is all-or-nothing
- **Consistency**: Database remains consistent before and after transaction execution
- **Isolation**: Concurrent transactions are isolated from each other
- **Durability**: Committed transactions are permanently saved

### Isolation Levels
1. **Read Uncommitted**: Allow reading uncommitted data (dirty reads)
2. **Read Committed**: Only read committed data (non-repeatable reads)
3. **Repeatable Read**: Multiple reads in same transaction return same results (phantom reads)
4. **Serializable**: Full isolation, highest level

### Distributed Transactions
Two-Phase Commit Protocol (2PC):
- **Prepare Phase**: Coordinator asks participants if they can commit
- **Commit Phase**: If all participants agree, coordinator sends commit command

### Optimistic vs Pessimistic Locking
- **Optimistic Locking**: Assume conflicts are rare, check version at commit
- **Pessimistic Locking**: Assume conflicts are common, lock resources first

### Deadlock Handling
1. **Timeout**: Rollback after timeout
2. **Deadlock Detection**: Periodically detect and choose victim
3. **Prevention**: Acquire resources in order
`,

	`## Network Security Fundamentals

Network security is the practice of protecting networks and data from attacks.

### Common Attack Types
1. **SQL Injection**: Inject malicious SQL statements to retrieve data
2. **XSS (Cross-Site Scripting)**: Inject malicious scripts into web pages
3. **CSRF (Cross-Site Request Forgery)**: Trick users into unintended actions
4. **DDoS**: Distributed denial of service attacks

### Protection Measures
- **Input Validation**: Check and filter all user input
- **Parameterized Queries**: Prevent SQL injection
- **HTTPS**: Encrypt data in transit
- **CSP (Content Security Policy)**: Prevent XSS
- **CSRF Token**: Prevent cross-site request forgery

### Encryption Technologies
Symmetric encryption:
- AES: Efficient, suitable for large amounts of data
- Key distribution is the challenge

Asymmetric encryption:
- RSA: Public key encryption, private key decryption
- Suitable for key exchange and digital signatures

Hash functions:
- SHA-256: One-way hash
- Used for data integrity verification

### Authentication
- **MFA (Multi-Factor Authentication)**: Increased security
- **OAuth 2.0**: Third-party authorization
- **JWT**: Stateless authentication tokens

### Security Best Practices
1. Principle of least privilege
2. Defense in depth
3. Regular security audits
4. Timely patch management
5. Secure coding standards
`,

	`## Containerization and Microservices Architecture

Container technology has changed application deployment.

### Docker Core Concepts
1. **Image**: Read-only application template
2. **Container**: Running instance of an image
3. **Registry**: Store and distribute images

### Dockerfile Best Practices
# Use multi-stage builds to reduce image size
FROM python:3.12 AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --user -r requirements.txt

FROM python:3.12-slim
COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH
WORKDIR /app
COPY . .
CMD ["python", "main.py"]

### Kubernetes Orchestration
Kubernetes provides container orchestration:
- **Pod**: Smallest deployment unit
- **Service**: Service discovery and load balancing
- **Deployment**: Declarative deployment management
- **ConfigMap/Secret**: Configuration management

### Microservice Design Principles
1. **Single Responsibility**: Each service focuses on one business function
2. **Stateless**: Services don't maintain session state
3. **API First**: Services communicate via APIs
4. **Independent Deployment**: Services can be updated independently

### Service Communication
- **REST**: Simple and universal
- **gRPC**: High performance, based on Protobuf
- **Message Queue**: Async decoupling

### Observability
- **Logging**: Structured log output
- **Metrics**: Prometheus for metrics collection
- **Tracing**: Jaeger for distributed tracing
`,

	`## Redis Caching Strategies

Redis is a high-performance in-memory data structure store.

### Data Structures
1. **String**: Key-value pairs, stores strings, numbers
2. **Hash**: Field-value pairs collections
3. **List**: Ordered string list
4. **Set**: Unordered unique string collection
5. **Sorted Set**: Ordered set with scores

### Caching Patterns
1. **Cache-Aside**:
   - Check cache first, on miss fetch from DB and update cache
   - Write directly to DB, invalidate cache

2. **Write-Through**:
   - Update both cache and DB on write
   - Ensures cache consistency

3. **Write-Behind**:
   - Update cache only, async batch write to DB
   - High performance but potential data loss

### Cache Eviction Strategies
- **TTL**: Set expiration time
- **LRU**: Least recently used eviction
- **LFU**: Least frequently used eviction

### Distributed Cache Issues
1. **Cache Penetration**: Query for non-existent data
   - Solution: Bloom filter

2. **Cache Breakdown**: Hot key expires
   - Solution: Mutex update

3. **Cache Avalanche**: Many keys expire simultaneously
   - Solution: Random TTL

### Redis Cluster
- **Master-Slave Replication**: Read-write separation
- **Sentinel**: Automatic failover
- **Cluster**: Data sharding
`,

	`## Message Queue Design

Message queues enable async processing and service decoupling.

### Message Queue Patterns
1. **Point-to-Point**: One-to-one consumption
2. **Publish-Subscribe**: One-to-many broadcast

### Message Delivery Guarantees
- **At Most Once**: May lose, no duplicates
- **At Least Once**: May duplicate, no loss
- **Exactly Once**: No loss, no duplicates (ideal)

### Common Message Queues
1. **RabbitMQ**: AMQP protocol, feature-rich
2. **Kafka**: High throughput, log collection
3. **Redis**: Lightweight, simple scenarios

### Dead Letter Queue
Handle messages that cannot be consumed normally:
- Message retry exceeds maximum attempts
- Queue reaches maximum length
- Message is rejected and not re-queued

### Message Order Guarantees
- Single partition: Ordered within partition
- Consumer groups: Ordered within consumer group
- Sequence numbers: Application-layer ordering

### Message Idempotency
Prevent issues from duplicate consumption:
- Unique message ID deduplication
- Idempotent business operation design
`,
}

// generateQueryContent generates query content by repeating paragraphs until target size is reached.
func generateQueryContent(targetTokens int, paragraphIndex int) (string, int) {
	var content string
	var estimatedTokens int
	contentIndex := paragraphIndex

	for estimatedTokens < targetTokens {
		paragraph := longTextParagraphs[contentIndex%len(longTextParagraphs)]
		contentIndex++

		// Rough token estimation: 1 token ≈ 4 characters for English text
		paragraphTokens := len(paragraph) / 4
		content += "\n\n" + paragraph
		estimatedTokens += paragraphTokens
	}

	return content, estimatedTokens
}
