package watcher

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ruffini/prism/parser"
)

// IndexFn es la función que reindexea un único archivo
type IndexFn func(path string) error

// RemoveFn elimina los nodos de un archivo del grafo
type RemoveFn func(path string) error

// Watch observa el directorio root y llama a indexFn/removeFn ante cambios.
// Bloquea hasta que se cierre done.
func Watch(root string, indexFn IndexFn, removeFn RemoveFn, done <-chan struct{}) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := w.Add(root); err != nil {
		return err
	}

	log.Printf("Watching %s for changes...", root)

	debounce := make(map[string]*time.Timer)

	for {
		select {
		case <-done:
			return nil

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			path := event.Name

			// Solo procesar archivos de código
			if !parser.IsCodeFile(path) {
				continue
			}

			// Debounce: esperar 500ms antes de procesar
			if t, exists := debounce[path]; exists {
				t.Stop()
			}

			op := event.Op
			debounce[path] = time.AfterFunc(500*time.Millisecond, func() {
				delete(debounce, path)
				if op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0 {
					if err := removeFn(path); err != nil {
						log.Printf("Failed to remove nodes for %s: %v", path, err)
					} else {
						log.Printf("Removed nodes for %s", path)
					}
					return
				}
				if op&(fsnotify.Create|fsnotify.Write) != 0 {
					if err := indexFn(path); err != nil {
						log.Printf("Failed to reindex %s: %v", path, err)
					} else {
						log.Printf("Reindexed %s", path)
					}
				}
			})

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
