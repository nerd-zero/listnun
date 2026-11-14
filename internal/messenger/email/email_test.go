package email

import (
	"errors"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

// Push picked a server with rand.Intn(len(pool)), which panics on an empty
// pool, inside a campaign worker, taking the whole process down.
func TestNoServers(t *testing.T) {
	if _, err := New("email"); !errors.Is(err, ErrNoServers) {
		t.Errorf("New with no servers: err = %v, want ErrNoServers", err)
	}

	e := &Emailer{pools: map[string][]*Server{}}
	if err := e.Push(models.Message{To: []string{"a@example.com"}}); !errors.Is(err, ErrNoServers) {
		t.Errorf("Push with no servers: err = %v, want ErrNoServers", err)
	}
}
