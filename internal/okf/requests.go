package okf

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RequestsFile is the file in the bundle root that holds change requests.
// It is not markdown, so it is not a concept.
const RequestsFile = "requests.json"

// Request statuses.
const (
	RequestOpen = "open" // waiting for someone to act on it
	RequestDone = "done" // acted on; the concept wants another review
)

// Request is a reviewer's request to change a concept, and what was done
// about it.
type Request struct {
	ID string `json:"id"`
	// Concept is the bundle path of the concept, for example
	// "packages/app.md", or empty for a request about the bundle as a whole.
	Concept string    `json:"concept,omitempty"`
	Text    string    `json:"text"`
	By      string    `json:"by"`
	At      time.Time `json:"at"`
	Status  string    `json:"status"`

	// Set when the request is done.
	Response   string     `json:"response,omitempty"`
	ResolvedBy string     `json:"resolvedBy,omitempty"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
}

// Open reports whether the request is still waiting to be acted on.
func (r Request) Open() bool {
	return r.Status == RequestOpen
}

// LoadRequests reads the change requests of the bundle at root, oldest
// first. A bundle without a requests file has none.
func LoadRequests(root string) ([]Request, error) {
	raw, err := os.ReadFile(filepath.Join(root, RequestsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var reqs []Request
	if err := json.Unmarshal(raw, &reqs); err != nil {
		return nil, fmt.Errorf("%s: %w", RequestsFile, err)
	}
	return reqs, nil
}

// SaveRequests writes the change requests of the bundle at root, replacing
// the file in one step. With no requests left the file is removed.
func SaveRequests(root string, reqs []Request) error {
	file := filepath.Join(root, RequestsFile)
	if len(reqs) == 0 {
		err := os.Remove(file)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	data, err := json.MarshalIndent(reqs, "", "  ")
	if err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o664); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// AddRequest appends an open request and returns it with its new id. Ids
// are "r1", "r2" and so on, short enough to say to an agent.
func AddRequest(reqs []Request, concept, text, by string, at time.Time) ([]Request, Request) {
	highest := 0
	for _, r := range reqs {
		if n, err := strconv.Atoi(strings.TrimPrefix(r.ID, "r")); err == nil && n > highest {
			highest = n
		}
	}
	req := Request{
		ID: "r" + strconv.Itoa(highest+1), Concept: concept, Text: text,
		By: by, At: at.UTC().Truncate(time.Second), Status: RequestOpen,
	}
	return append(reqs, req), req
}

// ResolveRequest marks the open request with the given id as done, with a
// response saying what was changed.
func ResolveRequest(reqs []Request, id, response, by string, at time.Time) error {
	if strings.TrimSpace(response) == "" {
		return errors.New("a response saying what was changed is required")
	}
	for i := range reqs {
		if reqs[i].ID != id {
			continue
		}
		if !reqs[i].Open() {
			return fmt.Errorf("request %s is already %s", id, reqs[i].Status)
		}
		resolved := at.UTC().Truncate(time.Second)
		reqs[i].Status, reqs[i].Response, reqs[i].ResolvedBy, reqs[i].ResolvedAt = RequestDone, response, by, &resolved
		return nil
	}
	return fmt.Errorf("no request %s", id)
}

// WithdrawRequest removes the open request with the given id. A request
// that is already done stays, as a record of why a concept changed.
func WithdrawRequest(reqs []Request, id string) ([]Request, error) {
	for i, r := range reqs {
		if r.ID != id {
			continue
		}
		if !r.Open() {
			return reqs, fmt.Errorf("request %s is already %s", id, r.Status)
		}
		return append(reqs[:i:i], reqs[i+1:]...), nil
	}
	return reqs, fmt.Errorf("no request %s", id)
}
