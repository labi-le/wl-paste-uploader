.DEFAULT: run

PACKAGE_NAME = wl-uploader

MAIN_PATH = main.go
BUILD_PATH = build/package/
INSTALL_DIR = /usr/local/bin/

export CGO_ENABLED = 0

run:
	@go run $(MAIN_PATH)

build-release:
	@go build -ldflags "-w" -a -v -o $(BUILD_PATH)$(PACKAGE_NAME) $(MAIN_PATH)

build-dev:
	@go build -v -o $(BUILD_PATH)$(PACKAGE_NAME) $(MAIN_PATH)

install: build-release
	sudo cp $(BUILD_PATH)$(PACKAGE_NAME) $(INSTALL_DIR)$(PACKAGE_NAME)

uninstall:
	sudo rm $(INSTALL_DIR)$(PACKAGE_NAME)