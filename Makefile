.PHONY: setup start stop update logs core-logs ui-logs \
       build-core build-ui build-scanner \
       restart-core restart-ui restart-scanner

# --- Full Stack ---

setup:
	@echo "Initializing project..."
	bash scripts/setup.sh

start:
	@echo "Starting all services..."
	docker compose up -d

stop:
	@echo "Stopping all services..."
	docker compose down

update:
	@echo "Pulling latest..."
	git pull --ff-only

# --- Logs ---

logs:
	docker compose logs -f

core-logs:
	docker compose logs -f core

ui-logs:
	docker compose logs -f ui

scanner-logs:
	docker compose logs -f scanner


# --- Per-service build & restart ---

build-core:
	docker compose build core

build-ui:
	docker compose build ui


build-scanner:
	docker compose build scanner

restart-core:
	docker compose up -d --build core

restart-ui:
	docker compose up -d --build ui


restart-scanner:
	docker compose up -d --build scanner
