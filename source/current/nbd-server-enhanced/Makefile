# Enhanced NBD Server with SIGHUP Cache Flush
# MigrateKit OSSEA Enhancement for Multi-Disk Correlation

CC = gcc
CFLAGS = -Wall -Wextra -O2 -DHAVE_CONFIG_H
LIBS = -lglib-2.0 -lpthread

# Get GLib cflags and libs
GLIB_CFLAGS = $(shell pkg-config --cflags glib-2.0)
GLIB_LIBS = $(shell pkg-config --libs glib-2.0)

CFLAGS += $(GLIB_CFLAGS)
LIBS += $(GLIB_LIBS)

SOURCES = nbd-server.c
TARGET = nbd-server-enhanced-migratekit

all: $(TARGET)

$(TARGET): $(SOURCES) nbdsrv.h
	$(CC) $(CFLAGS) -o $(TARGET) $(SOURCES) $(LIBS)
	@echo "✅ Enhanced NBD server built successfully"
	@echo "🎯 Features: SIGHUP cache flush for multi-disk correlation"

clean:
	rm -f $(TARGET)

install: $(TARGET)
	sudo cp $(TARGET) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(TARGET)
	@echo "✅ Enhanced NBD server installed to /usr/local/bin/$(TARGET)"

.PHONY: all clean install








