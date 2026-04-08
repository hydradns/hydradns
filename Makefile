.PHONY: setup start stop update logs core-logs ui-logs \
       build-core build-ui build-landing build-scanner \
       restart-core restart-ui restart-landing restart-scanner

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
	@echo "Updating all submodules..."
	git submodule update --remote --merge

# --- Logs ---

logs:
	docker compose logs -f

core-logs:
	docker compose logs -f core

ui-logs:
	docker compose logs -f ui

scanner-logs:
	docker compose logs -f scanner

landing-logs:
	docker compose logs -f landing

# --- Per-service build & restart ---

build-core:
	docker compose build core

build-ui:
	docker compose build ui

build-landing:
	docker compose build landing

build-scanner:
	docker compose build scanner

restart-core:
	docker compose up -d --build core

restart-ui:
	docker compose up -d --build ui

restart-landing:
	docker compose up -d --build landing

restart-scanner:
	docker compose up -d --build scanner
