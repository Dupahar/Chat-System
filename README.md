# 🚀 Distributed Real-Time Chat System

> A high-performance, scalable, and fault-tolerant chat platform engineered for millions of concurrent users. Built with **Go**, **Kafka**, **ScyllaDB**, and **Next.js**.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Kafka-Redpanda-black?logo=apachekafka&logoColor=white)
![ScyllaDB](https://img.shields.io/badge/ScyllaDB-NoSQL-4495D1?logo=scylladb&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-13+-black?logo=next.js&logoColor=white)

## 📖 Overview

This project is a production-grade implementation of a modern chat infrastructure. Unlike simple chat apps that use a single server, this system is designed as a **distributed microservices architecture**. It decouples connection handling from message processing, allowing independent scaling of components.

The system leverages **Kafka** for asynchronous message routing, **ScyllaDB** for high-throughput message persistence, and **Redis** for real-time presence and ephemeral state.

## 🏗️ Architecture

The system is composed of four main services:

### 1. Gateway Service (Go)
- **Role**: Connection Terminator & Event Broadcaster.
- **Responsibilities**:
  - Handles thousands of concurrent WebSocket connections.
  - Manages real-time user presence using **Redis Sets**.
  - Broadcasts messages to connected clients via internal Go channels.
  - Forwards incoming messages to **Kafka** for processing.

### 2. Messaging Service (Go)
- **Role**: Async Worker & Persister.
- **Responsibilities**:
  - Consumes messages from **Kafka** topics.
  - Persists chat history to **ScyllaDB** (optimized for write-heavy workloads).
  - Updates conversation metadata (unread counts, last message pointers).
  - Handles "fan-out" logic for group chats (future proofing).

### 3. API Service (Go)
- **Role**: REST Interface.
- **Responsibilities**:
  - Handles User Authentication (JWT).
  - Serves chat history with pagination.
  - Manages conversation lists and read receipts.
  - Provides presence snapshots.

### 4. Frontend (Next.js / TypeScript)
- **Role**: User Interface.
- **Features**:
  - Modern, responsive UI built with **Tailwind CSS**.
  - Real-time updates via WebSockets.
  - **Markdown Support**: Rich text rendering for messages.
  - **Smart Notifications**: Unread badges and read receipts (✓/✓✓).

---

## ✨ Key Features

- **⚡ Real-Time Messaging**: Instant delivery with low latency.
- **🔒 Secure**: JWT-based authentication for all connections.
- **💾 Persistent History**: Messages are stored safely in ScyllaDB.
- **👀 Presence System**: Real-time "Online" status indicators.
- **💬 Direct Messages**: Private 1-on-1 conversations.
- **🔴 Unread Badges**: Track unread messages per conversation.
- **✅ Read Receipts**: Know exactly when your message is read.
- **📝 Rich Text**: Support for **Markdown**, code blocks, and formatting.
- **🐳 Dockerized**: Complete environment setup with a single command.

---

## 🛠️ Tech Stack

| Component | Technology | Reason |
|-----------|------------|--------|
| **Backend** | Go (Golang) | High concurrency, low latency, efficient resource usage. |
| **Frontend** | Next.js, TypeScript | Type safety, component reusability, modern DX. |
| **Broker** | Redpanda (Kafka) | High-throughput event streaming, simpler than JVM Kafka. |
| **Database** | ScyllaDB | Drop-in Cassandra replacement, ultra-low latency writes. |
| **Cache** | Redis | Fast in-memory access for presence and ephemeral data. |
| **DevOps** | Docker Compose | Orchestration of the entire stack for development. |

---

## 🚀 Getting Started

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (optional, for local dev)
- Node.js 18+ (optional, for local dev)

### Quick Start (Docker)

Run the entire system with one command:

```bash
docker-compose up --build
```

Access the application at **http://localhost:3000**.

### Manual Setup

1. **Start Infrastructure**:
   ```bash
   docker-compose up -d redpanda scylladb redis
   ```

2. **Run Services**:
   ```bash
   # Terminal 1
   go run apps/gateway/main.go

   # Terminal 2
   go run apps/messaging/main.go

   # Terminal 3
   go run apps/api/main.go
   ```

3. **Run Frontend**:
   ```bash
   cd apps/web
   npm install
   npm run dev
   ```

---

## 🔮 Future Roadmap

- [ ] **Group Chats**: Support for multi-user channels.
- [ ] **Media Sharing**: Image and file uploads via S3/MinIO.
- [ ] **Push Notifications**: Mobile alerts using FCM.
- [ ] **Search**: Full-text message search with Elasticsearch.
- [ ] **E2EE**: End-to-end encryption for private chats.

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## � Author

**Adil Mahajan**
- Email: [mahajanadil0220@gmail.com](mailto:mahajanadil0220@gmail.com)
- GitHub: [@Dupahar](https://github.com/Dupahar)

## �📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
