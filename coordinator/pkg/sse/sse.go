package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
)

// SSEMessage type representing one SSE message.
type SSEMessage struct {
	ID    string // Optional unique identifier of the message.
	Event string // Event of the message. Used to distinguish messages that should be processed differently.
	Data  any    // JSON serializable value. Interface, so should always point to the newest data and doesn't copy.
}

// Hub type used for subscriber/publisher messaging.
type Hub struct {
	subs             map[string]map[uuid.UUID]chan SSEMessage // Maps storing SSE channels for unique topic and id combination.
	reg              chan sub                                 // Channel that holds channel registration to a topic requests.
	unreg            chan uuid.UUID                           // Channel that holds ids to be unregistered.
	pub              chan pubMsg                              // Channel that holds messages to be published.
	msgChannelSize   int                                      // Size of the message publication channel.
	retryInterval    int                                      // Interval of retrying SSE connection in milliseconds.
	keepaliveInteval time.Duration                            // Duration between keep-alive messages.
}

// sub type representing a subscription to a topic.
type sub struct {
	id     uuid.UUID       // Unique identifier of the subscription.
	topics []string        // Topics of the subscription.
	ch     chan SSEMessage // Channel to which send published messages from all the topics.
}

// pubMsg type represents SSE message published to a certain topic.
type pubMsg struct {
	topic string     // Topic for the message to be published to.
	msg   SSEMessage // SSE message to be published.
}

// NewHub creates and runs a new Hub
func NewHub() *Hub {
	h := &Hub{
		subs:             make(map[string]map[uuid.UUID]chan SSEMessage),
		reg:              make(chan sub),
		unreg:            make(chan uuid.UUID),
		pub:              make(chan pubMsg, 512),
		msgChannelSize:   16,
		retryInterval:    3000,
		keepaliveInteval: 20 * time.Second,
	}
	go h.run()
	return h
}

// run handles registration, unregistration, and publishing of messages.
func (h *Hub) run() {
	for {
		select {
		case s := <-h.reg:
			// Worker is registered
			for _, topic := range s.topics {
				m := h.subs[topic]
				if m == nil {
					m = make(map[uuid.UUID]chan SSEMessage)
					h.subs[topic] = m
				}
				m[s.id] = s.ch
			}
		case id := <-h.unreg:
			// Worker is unregistered
			for _, m := range h.subs {
				ch, ok := m[id]
				if ok {
					delete(m, id)
					close(ch)
				}
			}
		case p := <-h.pub:
			// Message is published
			for _, ch := range h.subs[p.topic] {
				select {
				case ch <- p.msg:
					// Sent message to each channel in the topic
				default:
				}
			}
		}
	}
}

// Publish publishes message to the given topic.
func (h *Hub) Publish(topic string, msg SSEMessage) {
	select {
	case h.pub <- pubMsg{topic, msg}:
	default:
	}
}

// PublishAll publishes message to all the given topics.
func (h *Hub) PublishAll(topics []string, msg SSEMessage) {
	for _, t := range topics {
		h.Publish(t, msg)
	}
}

// ServeSSE serves SSE subscriber of the given topics.
// c - gin context
// topics - slice of topics to subscribe to
// load - function that returns the first message to be sent
func (h *Hub) ServeSSE(c *gin.Context, topics []string, load func() ([]SSEMessage, bool)) {
	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Get a flusher to flush manually
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Create message channel, generate id, and register to all topics
	ch := make(chan SSEMessage, h.msgChannelSize)
	id := uuid.New()
	h.reg <- sub{id: id, topics: topics, ch: ch}
	defer func() { h.unreg <- id }()

	// Set retry interval and send initial message
	fmt.Fprint(c.Writer, fmt.Sprintf("retry: %d\n\n", h.retryInterval))
	if msgs, ok := load(); ok {
		for _, msg := range msgs {
			writeSSE(c.Writer, msg)
		}
	}
	flusher.Flush()

	// Create a ticker for keep-alive messages
	ticker := time.NewTicker(h.keepaliveInteval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			// Exit if request is closed
			return
		case <-ticker.C:
			// Send keep-alive message on a ticker, just a comment
			fmt.Fprint(c.Writer, ":\n\n")
			flusher.Flush()
		case msg, ok := <-ch:
			// Send and flush a message
			if !ok {
				return
			}
			writeSSE(c.Writer, msg)
			flusher.Flush()
		}
	}
}

// writeSSE writes SSEMesage to a http.ResponseWriter.
func writeSSE(w http.ResponseWriter, m SSEMessage) {
	if m.ID != "" {
		fmt.Fprintf(w, "id: %s\n", m.ID)
	}
	if m.Event != "" {
		fmt.Fprintf(w, "event: %s\n", m.Event)
	}
	payload, _ := json.Marshal(m.Data)
	fmt.Fprintf(w, "data: %s\n\n", payload)
}
