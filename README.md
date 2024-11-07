# RTMB - Real-Time Message Broker

RTMB is a lightweight, high-performance, real-time message broker written in Go. It implements a publish-subscribe (pub-sub) messaging pattern similar to NATS, allowing clients to communicate asynchronously by publishing messages to subjects and subscribing to them.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Usage](#usage)
  - [Server Setup](#server-setup)
  - [Client Interaction](#client-interaction)
    - [Available Commands](#available-commands)
    - [Command Reference](#command-reference)
      - [CONNECT](#connect)
      - [PING / PONG](#ping--pong)
      - [SUB](#sub)
      - [UNSUB](#unsub)
      - [PUB](#pub)
  - [Example Usage](#example-usage)
- [Code Structure](#code-structure)
  - [Commands Package](#commands-package)
  - [Topic Package](#topic-package)
    - [Locking Mechanism](#locking-mechanism)
  - [Client Connection](#client-connection)
- [Real-Time Messaging](#real-time-messaging)
- [Comparison with Other Message Brokers](#comparison-with-other-message-brokers)
  - [RTMB vs. RabbitMQ](#rtmb-vs-rabbitmq)
  - [RTMB vs. Apache Kafka](#rtmb-vs-apache-kafka)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Publish-Subscribe Messaging**: Clients can publish messages to subjects and subscribe to them to receive messages.
- **Subject Hierarchy with Wildcards**: Supports hierarchical subjects with single-level (`*`) and multi-level (`>`) wildcards for flexible subscription patterns.
- **Real-Time Communication**: Designed for low-latency, real-time messaging without persisting data to disk.
- **Concurrency Support**: Handles multiple clients concurrently with proper synchronization.
- **Thread-Safe**: Implements a consistent locking mechanism to ensure thread safety.
- **Lightweight and Efficient**: Uses efficient data structures and minimizes locking overhead for high performance.
- **Extensive Testing**: Includes comprehensive unit tests covering various scenarios.

## Architecture

RTMB consists of a server that handles client connections, parses incoming commands, and manages message delivery using a trie-based topic system. The main components are:

- **Server**: Accepts client connections and spawns goroutines to handle each client.
- **Commands Package**: Parses and handles commands sent by clients.
- **Topic Package**: Manages subjects, subscriptions, and message delivery.
- **Client Connection**: Represents a client connection and handles message sending.

## Installation

To build and run RTMB, you need to have Go installed (version 1.16 or later is recommended).

1. **Clone the repository:**

   ```bash
   git clone https://github.com/yourusername/rtmb.git
   cd rtmb
   ```

2. **Build the server:**

   ```bash
   go build -o rtmb
   ```

## Usage

### Server Setup

Run the RTMB server:

```bash
./rtmb
```

By default, the server listens on port `8080`. You can specify a different port using the `-port` flag:

```bash
./rtmb -port 4222
```

### Client Interaction

Clients can connect to the RTMB server using TCP and communicate using simple text-based commands. You can use tools like `telnet`, `nc` (netcat), or write your own client in any programming language that supports TCP sockets.

#### Available Commands

- **CONNECT**: Establish a connection with the server.
- **PING** / **PONG**: Heartbeat mechanism to keep the connection alive.
- **SUB**: Subscribe to a subject to receive messages.
- **UNSUB**: Unsubscribe from a subject.
- **PUB**: Publish a message to a subject.

#### Command Reference

Below is a detailed description of each command, including its syntax and usage.

##### CONNECT

Establishes a connection with the server.

- **Syntax:**

  ```
  CONNECT {}
  ```

- **Description:**

  - The `CONNECT` command initiates a connection with the server.
  - The `{}` represents an optional JSON payload for future extensions but is currently unused.

- **Example:**

  ```
  CONNECT {}
  ```

- **Server Response:**

  - The server may send an `INFO` message upon successful connection.

##### PING / PONG

Heartbeat mechanism to keep the connection alive.

- **PING**

  - **Syntax:**

    ```
    PING
    ```

  - **Description:**

    - The client sends a `PING` to the server to check if the connection is still alive.
    - The server should respond with a `PONG`.

- **PONG**

  - **Syntax:**

    ```
    PONG
    ```

  - **Description:**

    - Sent by either the client or the server in response to a `PING`.
    - Keeps the connection active.

- **Example:**

  ```
  PING
  ```

  - Server Response:

    ```
    PONG
    ```

##### SUB

Subscribe to a subject to receive messages.

- **Syntax:**

  ```
  SUB <subject> <sid>
  ```

  - `<subject>`: The subject to subscribe to (e.g., `foo.bar`).
  - `<sid>`: A unique subscription identifier (string).

- **Description:**

  - The `SUB` command registers the client to receive messages published to the specified subject.
  - Supports wildcards (`*` for single-level, `>` for multi-level).

- **Example:**

  ```
  SUB foo.bar sid1
  ```

- **Server Response:**

  - The server does not send a response upon successful subscription.

##### UNSUB

Unsubscribe from a subject.

- **Syntax:**

  ```
  UNSUB <sid> [max_msgs]
  ```

  - `<sid>`: The subscription ID to unsubscribe.
  - `[max_msgs]` (optional): If provided, the server will unsubscribe the client after sending `max_msgs` additional messages.

- **Description:**

  - The `UNSUB` command removes the subscription associated with the given `sid`.
  - If `max_msgs` is specified, the server will unsubscribe the client after `max_msgs` more messages are sent.

- **Examples:**

  - Unsubscribe immediately:

    ```
    UNSUB sid1
    ```

  - Unsubscribe after receiving 5 more messages:

    ```
    UNSUB sid1 5
    ```

- **Server Response:**

  - The server does not send a response upon successful unsubscription.

##### PUB

Publish a message to a subject.

- **Syntax:**

  ```
  PUB <subject> <msg_len>
  <message>
  ```

  - `<subject>`: The subject to publish the message to.
  - `<msg_len>`: The length of the message payload in bytes.
  - `<message>`: The message payload.

- **Description:**

  - The `PUB` command publishes a message to the specified subject.
  - The message payload follows the command, separated by a newline.
  - The payload length must match `<msg_len>`.

- **Example:**

  ```
  PUB foo.bar 11
  Hello World
  ```

  - In this example, `Hello World` is the message payload, which is 11 bytes long.

- **Server Response:**

  - The server does not send a response upon successful publication.

#### Notes on Subjects and Wildcards

- **Subject Format:**

  - Subjects are strings separated by dots (`.`), forming a hierarchy (e.g., `foo.bar.baz`).
  - Each token in the subject must be a non-empty string.

- **Single-Level Wildcard (`*`):**

  - Matches any token at a single level in the subject hierarchy.
  - Example:
    - Subscription: `foo.*.baz`
    - Matches: `foo.bar.baz`, `foo.qux.baz`

- **Multi-Level Wildcard (`>`):**

  - Matches any number of tokens at or beyond its position.
  - Must be the last token in the subject.
  - Example:
    - Subscription: `foo.>`
    - Matches: `foo.bar`, `foo.bar.baz`, `foo.bar.baz.qux`

### Example Usage

Below is an example of how to use RTMB using `telnet`.

1. **Connect to the Server:**

   ```bash
   telnet localhost 8080
   ```

2. **Establish a Connection:**

   ```
   CONNECT {}
   ```

3. **Subscribe to a Subject:**

   ```
   SUB foo.bar sid1
   ```

4. **Publish a Message:**

   Open another terminal and connect to the server:

   ```bash
   telnet localhost 8080
   ```

   Then send:

   ```
   CONNECT {}
   PUB foo.bar 12
   Hello World!
   ```

5. **Receive the Message:**

   - The client subscribed to `foo.bar` will receive:

     ```
     MSG foo.bar sid1 12
     Hello World!
     ```

6. **Unsubscribe from a Subject:**

   ```
   UNSUB sid1
   ```

7. **Ping-Pong to Keep Alive:**

   ```
   PING
   ```

   - Server responds:

     ```
     PONG
     ```

## Code Structure

The codebase is organized into several packages:

- `main.go`: Entry point of the server.
- `commands`: Handles parsing and execution of client commands.
- `topic`: Manages subjects, subscriptions, and message delivery.
- `parser`: Parses client commands.
- `client_connection`: Manages individual client connections.

### Commands Package

The `commands` package is responsible for handling parsed commands from clients and interacting with the `topic` package to manage subscriptions and message delivery.

Key components:

- `Commander`: Struct that encapsulates state and methods to handle commands for a client.
- `HandleCommand`: Dispatches commands based on their names.
- Command Handlers: `handleConnect`, `handleSub`, `handlePub`, `handleUnsub`.
- Client ID Management: Assigns a unique client ID using UUIDs.

Example of the `Commander` struct:

```go
type Commander struct {
    conn       net.Conn
    topic      *topic.Topic
    clientID   string
    clientSubs map[string]string // map of SID to subject
    port       string
}
```

### Topic Package

The `topic` package implements a trie data structure to manage subjects and subscriptions. It handles adding/removing subscribers, matching subjects for message delivery, and ensures thread safety through a consistent locking mechanism.

#### Key Components

- **Trie Structure**: Represents subjects hierarchically, allowing efficient matching and wildcard support.
- **Subscription Management**: Adds and removes subscribers based on subjects and subscription IDs.
- **Message Delivery**: Matches published subjects to subscribers and delivers messages.

#### Locking Mechanism

To ensure thread safety, the `topic` package uses per-node locks (`sync.RWMutex`) for each node in the trie. This approach allows concurrent read access and safe modifications.

**Locking Strategy:**

- **Traversal and Modification**:
  - Locks are acquired in a consistent order: parent node is locked, then child node is locked before releasing the parent lock.
  - This prevents deadlocks and ensures thread safety.
- **Use of `defer`**:
  - Locks are unlocked using `defer` immediately after they are acquired for the final node.
- **Avoiding Deadlocks**:
  - By not holding multiple locks simultaneously and locking nodes in a consistent order, deadlocks are prevented.

**Example of Adding a Subscriber:**

```go
func (t *Topic) AddSubscriber(sub *Subscription) error {
    // Validate subject and initialize traversal
    parts := splitSubject(sub.Subject)
    current := t.root
    current.mu.Lock()

    for _, part := range parts {
        child, exists := current.children[part]
        if !exists {
            child = NewTrieNode()
            current.children[part] = child
        }
        // Lock the child before unlocking the parent
        child.mu.Lock()
        current.mu.Unlock()
        current = child
    }

    // At this point, current.mu is locked for the target node
    defer current.mu.Unlock()

    if _, exists := current.subscribers[sub.SID]; exists {
        return fmt.Errorf("subscription with SID %s already exists for subject %s", sub.SID, sub.Subject)
    }
    current.subscribers[sub.SID] = sub
    return nil
}
```

### Client Connection

The `ClientConnection` manages individual client connections, including sending messages asynchronously using a buffered channel.

Key features:

- **Asynchronous Message Sending**: Uses a write loop to send messages without blocking the main thread.
- **Thread Safety**: Uses a mutex to protect access to the connection when writing messages.
- **Graceful Shutdown**: Implements a `Close` method to properly shut down the connection.

Example of the `ClientConnection` struct:

```go
type ClientConnection struct {
    ID        string
    Conn      io.Writer
    mu        sync.Mutex
    writeChan chan *Message // Channel for async message writing
    done      chan struct{} // Channel to signal shutdown
}
```

## Real-Time Messaging

### What Does "Real-Time" Mean in RTMB?

In the context of RTMB, "real-time" refers to the system's ability to deliver messages with minimal latency, ensuring that subscribers receive messages as soon as they are published. This is achieved by:

- **In-Memory Operation**: RTMB operates entirely in memory and does not persist data to disk. This eliminates disk I/O overhead, resulting in faster message processing and delivery.
- **No Message Persistence**: Messages are not stored persistently. If the server restarts or crashes, messages that have not been delivered are lost. This design choice prioritizes low latency over durability.
- **Optimized for Speed**: The server uses efficient data structures and minimizes locking overhead to handle high-throughput, low-latency messaging.

### Implications of Non-Persistent Messaging

- **High Performance**: By not persisting messages to disk, RTMB can achieve higher throughput and lower latency compared to brokers that write messages to disk.
- **No Message Durability**: Messages are transient. If a subscriber is not connected at the time of publishing, it will not receive the message later.
- **Use Cases**: Suitable for applications where real-time data delivery is critical, and message loss in the event of failures is acceptable (e.g., live data feeds, real-time analytics, gaming).

## Comparison with Other Message Brokers

### RTMB vs. RabbitMQ

**RabbitMQ** is a robust message broker that supports multiple messaging protocols and offers features like message persistence, acknowledgments, and complex routing.

- **Message Persistence**:
  - **RabbitMQ**: Supports durable queues and message persistence to disk, ensuring messages are not lost if the broker restarts.
  - **RTMB**: Does not persist messages to disk, prioritizing low latency over durability.
- **Features**:
  - **RabbitMQ**: Offers a wide range of features, including transactions, message acknowledgments, and advanced routing with exchanges.
  - **RTMB**: Focuses on simplicity and high-speed pub-sub messaging without additional overhead.
- **Use Cases**:
  - **RabbitMQ**: Suitable for applications requiring reliable message delivery, complex routing, and where durability is important.
  - **RTMB**: Ideal for real-time applications where speed is critical, and occasional message loss is acceptable.

### RTMB vs. Apache Kafka

**Apache Kafka** is a distributed streaming platform designed for high-throughput, scalable messaging with strong durability guarantees.

- **Message Persistence**:
  - **Kafka**: Persists all messages to disk and replicates them across multiple brokers for fault tolerance.
  - **RTMB**: Operates in-memory without persisting messages.
- **Data Retention**:
  - **Kafka**: Allows configuring retention policies to keep messages for a specified duration or size.
  - **RTMB**: Messages are transient and only exist in memory until delivered.
- **Throughput and Latency**:
  - **Kafka**: Optimized for high throughput, capable of handling large volumes of data with some latency.
  - **RTMB**: Prioritizes low latency over throughput, delivering messages in real-time.
- **Use Cases**:
  - **Kafka**: Suitable for event streaming, log aggregation, and scenarios where message durability and ordering are essential.
  - **RTMB**: Best for real-time messaging needs where immediate delivery is more important than durability.

### Summary

| Feature             | RTMB                     | RabbitMQ                  | Apache Kafka                   |
| ------------------- | ------------------------ | ------------------------- | ------------------------------ |
| Message Persistence | No (in-memory only)      | Yes (optional)            | Yes (always)                   |
| Latency             | Low (real-time delivery) | Moderate                  | Moderate                       |
| Throughput          | High                     | Moderate                  | Very High                      |
| Durability          | No                       | Yes (configurable)        | Yes (built-in)                 |
| Complexity          | Simple                   | Moderate                  | Complex                        |
| Use Cases           | Real-time messaging      | Reliable message delivery | Event streaming and processing |

## Testing

The project includes a comprehensive set of unit tests in `topic/topic_test.go` that cover various scenarios:

- Adding subscribers
- Removing subscribers
- Matching and delivering messages
- Wildcard subscriptions
- Handling unsubscriptions with `MaxMsgs`
- Concurrent operations

### Running Tests

To run the tests, navigate to the project directory and execute:

```bash
go test -race ./...
```

Using the `-race` flag enables the race detector to catch any data races during concurrent operations.

### Example Test Case

```go
func TestMatchAndDeliver(t *testing.T) {
    topic := NewTopic()
    client1 := NewMockClientConnection("client1")
    client2 := NewMockClientConnection("client2")

    sub1 := &Subscription{
        ClientID: client1.ID,
        SID:      "sid1",
        Subject:  "foo.bar",
        Client:   client1,
    }
    sub2 := &Subscription{
        ClientID: client2.ID,
        SID:      "sid2",
        Subject:  "foo.*",
        Client:   client2,
    }

    err := topic.AddSubscriber(sub1)
    if err != nil {
        t.Errorf("Failed to add subscriber 1: %v", err)
    }
    err = topic.AddSubscriber(sub2)
    if err != nil {
        t.Errorf("Failed to add subscriber 2: %v", err)
    }

    msg := []byte("hello world")
    subscribers, err := topic.MatchAndDeliver("foo.bar", msg)
    if err != nil {
        t.Errorf("Failed to match and deliver: %v", err)
    }
    if len(subscribers) != 2 {
        t.Errorf("Expected 2 subscribers, got %d", len(subscribers))
    }

    // Check if clients received the message
    if len(client1.Messages) != 1 {
        t.Errorf("Client 1 should have received 1 message, got %d", len(client1.Messages))
    }
    if len(client2.Messages) != 1 {
        t.Errorf("Client 2 should have received 1 message, got %d", len(client2.Messages))
    }
}
```

## Contributing

Contributions are welcome! Please follow these steps:

1. **Fork the repository.**

2. **Create a new branch:**

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes and commit them:**

   ```bash
   git commit -m "Add your message here"
   ```

4. **Push to your fork:**

   ```bash
   git push origin feature/your-feature-name
   ```

5. **Open a pull request.**

Please ensure that your code adheres to the existing coding style and includes tests where appropriate.

## License

This project is licensed under the terms of the [MIT License](LICENSE).
