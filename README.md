Got it! Here's an updated version of the `README.md` to reflect that it's a **self-hosted, Docker-based notification service** rather than an SDK, and that **Makefile commands** are used to interact with it:

---

# 📣 Notifications Service

A self-hosted, lightweight notification system powered by **PostgreSQL** and **Kafka**. Easily plug this service into your existing applications to manage topic-based notifications, client subscriptions, and real-time event delivery.

---

## 🚀 Features

- Dockerized and ready for production
- Topic-based subscription model
- Built on PostgreSQL + Kafka
- Type-safe queries using `sqlc`
- Simple Makefile-based control interface

---

## ⚙️ Setup & Run

### 1. Clone the Repository

```bash
git clone https://github.com/CodegeassBala/notify.git
cd notifications-service
```

### 2. Configure Environment

Create a `.env` file or export the following env variables:

```env
DB_URL=postgres://<username>:<password>@localhost:5432/notify?sslmode=disable
KAFKA_BROKER=localhost:9092
```

You may also adjust config values inside the Makefile as needed.

---

### 3. Start the Application

Use the provided `Makefile` to start the app and dependencies:

```bash
make start         # Start all containers
make dev        # Run the app in dev mode      
make clean       # Stop all containers and processes
```

> Note: This will spin up PostgreSQL, Kafka (via KRaft or ZooKeeper), and the notifications service container.

---

## 📦 Usage

Once running, your other services can interact with the notifications service via exposed HTTP endpoints (or any messaging integration you’ve configured). Example flows include:

- **Create a Topic**
- **Subscribe Clients to Topics**
- **Send Notifications to Topics**
- **Receive Events in Real Time**

---

## 🧪 Development

- DB access and migrations handled via `sqlc`
- Uses Go modules, pgx (or lib/pq), and clean module separation
- Add your queries inside the `query.sql` and regenerate with:

```bash
make dev
```

---
