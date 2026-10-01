package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	TopicWorkers = "workers"
	TopicJobs    = "jobs"
)

func WorkerTopic(id string) string    { return "workers:" + id }
func JobTopic(id string) string       { return "jobs:" + id }
func JobEventsTopic(id string) string { return "jobs:" + id + ":events" }

type SSEMessage struct {
	Event string
	Data  interface{}
}

type Hub struct {
	subs  map[string]map[chan SSEMessage]struct{}
	reg   chan sub
	unreg chan sub
	pub   chan pubMsg
}

type sub struct {
	topic string
	ch    chan SSEMessage
}

type pubMsg struct {
	topic string
	msg   SSEMessage
}

func NewHub() *Hub {
	h := &Hub{
		subs:  make(map[string]map[chan SSEMessage]struct{}),
		reg:   make(chan sub),
		unreg: make(chan sub),
		pub:   make(chan pubMsg, 512),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case s := <-h.reg:
			m := h.subs[s.topic]
			if m == nil {
				m = make(map[chan SSEMessage]struct{})
				h.subs[s.topic] = m
			}
			m[s.ch] = struct{}{}
		case s := <-h.unreg:
			if m := h.subs[s.topic]; m != nil {
				if _, ok := m[s.ch]; ok {
					delete(m, s.ch)
					close(s.ch)
				}
			}
		case p := <-h.pub:
			for ch := range h.subs[p.topic] {
				select {
				case ch <- p.msg:
				default:
					// slow subscriber — drop the message, don't block the hub
				}
			}
		}
	}
}

func (h *Hub) Publish(topic string, msg SSEMessage) {
	select {
	case h.pub <- pubMsg{topic, msg}:
	default:
	}
}

func (h *Hub) PublishAll(topics []string, msg SSEMessage) {
	for _, t := range topics {
		h.Publish(t, msg)
	}
}

func (h *Hub) ServeSSE(c *gin.Context, topic string, load func() SSEMessage) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ch := make(chan SSEMessage, 16)
	h.reg <- sub{topic: topic, ch: ch}
	defer func() { h.unreg <- sub{topic: topic, ch: ch} }()

	fmt.Fprint(c.Writer, "retry: 3000\n\n")
	if load != nil {
		writeSSE(c.Writer, load())
	}
	flusher.Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprint(c.Writer, ": keep-alive\n\n")
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(c.Writer, msg)
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, m SSEMessage) {
	if m.Event != "" {
		fmt.Fprintf(w, "event: %s\n", m.Event)
	}
	payload, _ := json.Marshal(m.Data)
	fmt.Fprintf(w, "data: %s\n\n", payload)
}
