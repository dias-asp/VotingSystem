SERVICES    := voting_service anonym_service audit_service auth_service
COMPOSE     := docker compose -f infrastrucrure/docker/docker-compose.yml
COMPOSE_DEV := docker compose -f infrastrucrure/docker/docker-compose.dev.yml

.PHONY: all build test tidy vet fmt lint compose-up compose-down compose-logs dev-up dev-down dev-logs dev-build clean

all: build

build: $(addprefix build-,$(SERVICES))
test:  $(addprefix test-,$(SERVICES))
tidy:  $(addprefix tidy-,$(SERVICES))

build-%:
	cd $* && go build -trimpath -o bin/$* ./cmd/$*

run-%:
	cd $* && go run ./cmd/$*

test-%:
	cd $* && go test ./...

tidy-%:
	cd $* && go mod tidy

vet:
	@for s in $(SERVICES); do (cd $$s && go vet ./...) || exit 1; done

fmt:
	@for s in $(SERVICES); do (cd $$s && gofmt -w .); done

lint: vet

compose-up:
	$(COMPOSE) up -d --build

compose-down:
	$(COMPOSE) down -v

compose-logs:
	$(COMPOSE) logs -f

dev-up:
	$(COMPOSE_DEV) up -d --build

dev-build:
	$(COMPOSE_DEV) build

dev-down:
	$(COMPOSE_DEV) down -v

dev-logs:
	$(COMPOSE_DEV) logs -f

clean:
	@for s in $(SERVICES); do rm -rf $$s/bin; done

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
