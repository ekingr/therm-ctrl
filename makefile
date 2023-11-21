dist := dist
deploy := ./deploy

.PHONY: all\
		prep\
		bin\
		cfg\
		install update

# All
all: prep bin cfg


# Preparation
prep: | $(dist)

$(dist):
	@mkdir -p "$@"


# Binary
bin: prep $(addprefix $(dist)/, \
		therm-ctrl_amd64\
		therm-ctrl_arm64\
		therm-ctrl_armv6\
		)

$(dist)/therm-ctrl_amd64: *.go
	go mod tidy
	go fmt
	go build -o "$@"

$(dist)/therm-ctrl_arm64: *.go
	go mod tidy
	go fmt
	GOOS=linux GOARCH=arm64 go build -o "$@"

$(dist)/therm-ctrl_armv6: *.go
	go mod tidy
	go fmt
	GOOS=linux GOARCH=arm GOARM=6 go build -o "$@"


# Configuration
cfg: $(dist)/init_systemd.conf $(dist)/init_upstart.conf

$(dist)/%.conf: %.conf
	cp "$<" "$@"


# Deployment
install: bin cfg
	$(deploy) INSTALL

update: bin cfg
	$(deploy) UPDATE

