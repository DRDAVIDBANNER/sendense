package main

import (
	"context"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	log.Println("🔍 Testing raw inotify events on /sys/block...")

	// Create a filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Watch /sys/block
	err = watcher.Add("/sys/block")
	if err != nil {
		log.Fatalf("Failed to watch /sys/block: %v", err)
	}

	log.Println("✅ Watching /sys/block for raw filesystem events...")
	log.Println("👀 Monitoring for 60 seconds - try attaching/detaching volumes now...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	eventCount := 0
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Println("Events channel closed")
				return
			}

			eventCount++
			log.Printf("🎯 Raw Event #%d:", eventCount)
			log.Printf("   Operation: %s", event.Op.String())
			log.Printf("   Path: %s", event.Name)
			log.Printf("   Time: %s", time.Now().Format("15:04:05.000"))

		case err, ok := <-watcher.Errors:
			if !ok {
				log.Println("Errors channel closed")
				return
			}
			log.Printf("❌ Watcher error: %v", err)

		case <-ctx.Done():
			log.Printf("🏁 Monitoring completed. Total events detected: %d", eventCount)
			return
		}
	}
}
