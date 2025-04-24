.PHONY: build clean install uninstall update

BINARY_NAME=tasks
INSTALL_DIR=/opt/tasks
DATA_DIR=/var/lib/tasks
SERVICE_FILE=tasks.service
SYSTEMD_DIR=/etc/systemd/system

build:
	go build -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

install: build
	# Create directories
	mkdir -p $(INSTALL_DIR)
	mkdir -p $(DATA_DIR)
	
	# Copy binary and set permissions
	cp $(BINARY_NAME) $(INSTALL_DIR)/
	chmod 755 $(INSTALL_DIR)/$(BINARY_NAME)
	
	# Copy service file and enable service
	cp $(SERVICE_FILE) $(SYSTEMD_DIR)/
	systemctl daemon-reload
	systemctl enable $(SERVICE_FILE)
	systemctl start $(SERVICE_FILE)
	
	@echo "Installation complete. Service is running on port 8085."

update: build
	# Stop the service before replacing the binary
	systemctl stop $(SERVICE_FILE)
	
	# Copy binary and set permissions
	cp $(BINARY_NAME) $(INSTALL_DIR)/
	chmod 755 $(INSTALL_DIR)/$(BINARY_NAME)
	
	# Start the service again
	systemctl start $(SERVICE_FILE)
	
	@echo "Update complete. Service has been restarted."

uninstall:
	# Stop and disable service
	systemctl stop $(SERVICE_FILE) || true
	systemctl disable $(SERVICE_FILE) || true
	
	# Remove files
	rm -f $(SYSTEMD_DIR)/$(SERVICE_FILE)
	rm -rf $(INSTALL_DIR)
	
	# Reload systemd
	systemctl daemon-reload
	
	@echo "Uninstallation complete."