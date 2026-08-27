package tunnel

import (
	"fmt"
	"sync"
	"time"
)

const logBufferCapacity = 500

type LogBuffer struct {
	mu    sync.Mutex
	lines [logBufferCapacity]string
	start int
	count int
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{}
}

func (b *LogBuffer) Add(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.count < logBufferCapacity {
		index := (b.start + b.count) % logBufferCapacity
		b.lines[index] = line
		b.count++
		return
	}
	b.lines[b.start] = line
	b.start = (b.start + 1) % logBufferCapacity
}

func (b *LogBuffer) AddAt(timestamp time.Time, message string) {
	b.Add(fmt.Sprintf("[%s] %s", timestamp.Format("15:04:05"), message))
}

func (b *LogBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, b.count)
	for index := range b.count {
		result[index] = b.lines[(b.start+index)%logBufferCapacity]
	}
	return result
}
