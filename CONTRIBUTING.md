# Contributing to HydraDNS

Thanks for your interest in contributing! HydraDNS is an open-source DNS security gateway and we welcome contributions of all kinds.

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose
- Make

### Development Setup

```bash
# Clone with submodules
git clone --recursive https://github.com/hydradns/hydradns.git
cd hydradns

# Start the full stack
make start

# Or work on individual services
cd apps/core && make build && make test
cd apps/ui && npm install && npm run dev
cd apps/cli && go build -o hydra .
```

## Repository Structure

This is a monorepo of Git submodules. Each service has its own repository:

| Service | Repo |
|:--------|:-----|
| Core | `hydradns/hydra-core` |
| Dashboard | `hydradns/hydra-ui` |
| Landing | `hydradns/hydradns-landing` |
| Scanner | `hydradns/scanner` |
| CLI | `hydradns/hydra-cli` |

Work inside each `apps/<service>` directory and push to that service's repo.

## Making Changes

1. Fork the relevant repository
2. Create a feature branch: `git checkout -b feature/my-change`
3. Make your changes
4. Run tests: `make test` (Go) or `npm run lint` (TypeScript)
5. Commit with a clear message describing the change
6. Push and open a pull request

## Code Style

### Go (Core, Scanner, CLI)

- Run `make fmt` before committing
- Run `make vet` and `make lint` (golangci-lint) to catch issues
- Write tests for new functionality
- Follow standard Go conventions (effective Go, Go Code Review Comments)

### TypeScript (Dashboard, Landing)

- Run `npm run lint` before committing
- Use TypeScript strict mode
- Prefer typed API responses over `any`

## What to Contribute

- Bug fixes (check issues for reported bugs)
- Test coverage improvements
- Documentation improvements
- New blocklist format parsers
- Dashboard UX improvements
- CLI commands for missing operations

## Reporting Issues

Open an issue on the relevant service repository with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Environment details (OS, Docker version, etc.)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
