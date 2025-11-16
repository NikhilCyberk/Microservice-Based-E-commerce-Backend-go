# Microservice-Based E-commerce Backend

This project is a microservices-based e-commerce backend built with Go, gRPC, and RabbitMQ.

## Architecture

- **API Gateway**: HTTP REST API gateway (port 8080)
- **User Service**: gRPC service for user management (port 50051)
- **Product Service**: gRPC service for product management (port 50052)
- **Order Service**: gRPC service for order management (port 50053)
- **RabbitMQ**: Message broker for event-driven communication (ports 5672, 15672)

## Prerequisites

- Docker Desktop installed and running
- Docker Compose (included with Docker Desktop)

## Running the Project

### Using Docker Compose (Recommended)

1. Make sure Docker Desktop is running
2. Run the following command from the project root:

```bash
docker-compose up --build
```

This will:
- Build all service Docker images
- Start RabbitMQ
- Start all microservices
- Start the API Gateway

### Accessing the Services

- **API Gateway**: http://localhost:8080
- **RabbitMQ Management UI**: http://localhost:15672 (username: admin, password: admin)

### API Endpoints

- `POST /users` - Create a user
- `POST /products` - Create a product
- `GET /products` - List all products
- `POST /orders` - Create an order

### Example API Calls

#### Create a User
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","name":"John Doe"}'
```

#### Create a Product
```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop","description":"High performance laptop","price":999.99,"stock":10}'
```

#### List Products
```bash
curl http://localhost:8080/products
```

#### Create an Order
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"<user_id>","items":[{"product_id":"<product_id>","quantity":2}]}'
```

## Stopping the Services

Press `Ctrl+C` in the terminal where docker-compose is running, or run:

```bash
docker-compose down
```

## Project Structure

```
.
├── api-gateway/          # API Gateway service
├── user-service/         # User microservice
├── product-service/      # Product microservice
├── order-service/        # Order microservice
└── docker-compose.yml    # Docker Compose configuration
```

## Development

Each service can be developed and tested independently. The services use in-memory storage, so data will be lost when services are restarted.

