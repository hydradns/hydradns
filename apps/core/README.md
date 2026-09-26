# HydraDNS 🛡️

Hactober Prerequisites

- Docker & Docker Compose
- Go 1.20 or higher (for development)
- Git

### Installation

1. **Clone the repository:**

   ```sh
   git clone https://github.com/lopster568/HydraDNS.git
   cd HydraDNS
   ```

2. **Configure the environment** (optional):

   ```sh
   # Copy the example config
   cp configs/config.yaml.example configs/config.yaml

   # Edit the configuration file as needed
   vim configs/config.yaml
   ```

3. **Build and run using Docker Compose:**
   ```sh
   docker-compose up --build
   ```

## 🔧 Usage

Once running, HydraDNS provides two main services:

### Data Plane (DNS Server)

- **Port**: 1053 (UDP and TCP)
- **Purpose**: Handles DNS queries with security filtering
- **Test it**: `dig @localhost -p 1053 example.com`

### Control Plane (Admin API)

- **Port**: 8086
- **Purpose**: Configuration and monitoring interface

[![Hacktoberfest 2025](https://img.shields.io/badge/Hacktoberfest-2025-orange.svg)](https://hacktoberfest.com)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)
[![Contributors Welcome](https://img.shields.io/badge/contributors-welcome-brightgreen.svg)](./CONTRIBUTING.md)

HydraDNS is a powerful DNS-layer security & privacy gateway designed to protect your network from threats while maintaining your privacy. Whether you're running it on a Raspberry Pi at home or deploying it in the cloud, HydraDNS has got you covered.

## ✨ Features

- 🔒 **DNS-layer Security**: Intercepts and filters DNS queries
- 🛡️ **Threat Protection**: Blocks malware, trackers, and unwanted ads
- 📊 **Detailed Reporting**: Monitor your network's security status
- 🎮 **CLI Administration**: Easy-to-use command line interface
- 🐳 **Container Ready**: Deploy anywhere with Docker support
- 🚀 **High Performance**: Optimized for both small devices and cloud deployments

## 🏗️ Architecture

HydraDNS uses a microservices architecture with two main components:

1. **Data Plane**: The core DNS server handling queries (port 1053)
2. **Control Plane**: Administrative API for configuration (port 8086)

Note: A file is not just code, it is a concept; and nothing should break the concept of a file, create separations.

## 🚀 Quick Start

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/lopster568/HydraDNS.git
    cd HydraDNS
    ```

2.  **Build and run the services using Docker Compose:**
    ```sh
    docker-compose up --build
    ```

## Usage

The control plane and data plane services will be running in the background.

- **Data Plane (DNS Server):** Listening on port `1053` (UDP and TCP).
- **Control Plane (Admin API):** Listening on port `8086`.

## 🛠️ Development

Want to contribute? Great! We use a standard Go project layout:

```
hydra-core/
├── cmd/                    # Main applications
│   ├── controlplane/      # Admin API service
│   └── dataplane/         # DNS server
├── internal/              # Private application code
│   ├── config/           # Configuration handling
│   ├── core/             # Core DNS logic
│   └── policy/           # Security policy engine
├── configs/               # Configuration files
└── docker/                # Dockerfiles
```

### Building from Source

```sh
# Build the data plane
go build -o bin/dataplane cmd/dataplane/main.go

# Build the control plane
go build -o bin/controlplane cmd/controlplane/main.go
```

### Running Tests

```sh
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🤝 Contributing

We welcome contributions! Check out our [Contributing Guidelines](./CONTRIBUTING.md) to get started.

### Good First Issues

Look for issues tagged with `good-first-issue` - these are perfect for newcomers!

### Getting Help

- 📖 Drop a mail @ rosh.s568@gmail.com
- 💬 Connect on [LinkedIn](https://www.linkedin.com/in/roshan-singh568)

## 📝 License

HydraDNS CE is licensed under the Apache License 2.0.  
See the [LICENSE](./LICENSE) file for details.

## ⭐ Show Your Support

If you find HydraDNS useful, please consider:

- Giving us a star on GitHub
- Contributing to the project
- Sharing it with others who might benefit

---

Built with ❤️ by the HydraDNS community
