# Transaction Service

RESTful API for managing cardholder accounts and financial transactions.

## Tech Stack

- **Go 1.26** — stdlib `net/http` router
- **MySQL 8.0** — relational database
- **Docker** — containerized application and database
- **Swagger** — interactive API documentation (swaggo)
- **Testify** — table-driven unit tests

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

No local Go or MySQL installation required — everything runs inside containers.

### Run

```bash
./run.sh
```

Or directly:

```bash
docker compose up --build
```

The API will be available at `http://localhost:3000`.

### Stop

```bash
docker compose down -v
```

## API Endpoints

| Method | Path                   | Description              |
|--------|------------------------|--------------------------|
| POST   | `/accounts`            | Create a new account     |
| GET    | `/accounts/{accountId}`| Get account by ID        |
| POST   | `/transactions`        | Create a new transaction |
| GET    | `/health`              | Health check             |

Full API documentation with request/response examples will be available at:

```
http://localhost:3000/swagger/index.html
```

## Testing

```bash
go test ./...
```