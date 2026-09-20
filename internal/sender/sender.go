package sender

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GhandyP/professional-email-drafting/internal/email"
)

// Message contains rendered draft content and its approval metadata.
type Message struct {
	DraftID    string
	To         string
	Subject    string
	Plain      string
	HTML       string
	ApprovedBy string
	ApprovedAt time.Time
}

// ApprovedMessage contains a message and its approval state.
type ApprovedMessage struct {
	message    Message
	authorized bool
}

// Message returns the held message by value.
func (m ApprovedMessage) Message() Message {
	return m.message
}

// Authorized reports whether the message is authorized for sending.
func (m ApprovedMessage) Authorized() bool {
	return m.authorized
}

// ErrSendingDisabled indicates that delivery is intentionally disabled.
var ErrSendingDisabled = errors.New("sending is disabled")

// ErrNotApproved indicates that a message lacks approval.
var ErrNotApproved = errors.New("message is not approved")

// Approved validates, authorizes, and renders a draft into an approved message.
func Approved(d email.Draft, in email.Input, approval email.Approval) (ApprovedMessage, error) {
	if err := d.Validate(in); err != nil {
		return ApprovedMessage{}, err
	}
	if err := approval.Authorizes(d); err != nil {
		return ApprovedMessage{}, err
	}

	html, err := d.RenderHTML()
	if err != nil {
		return ApprovedMessage{}, err
	}

	return ApprovedMessage{
		message: Message{
			DraftID:    approval.DraftID,
			To:         in.Recipient,
			Subject:    strings.TrimSpace(d.Subject),
			Plain:      d.RenderPlain(),
			HTML:       html,
			ApprovedBy: approval.ApprovedBy,
			ApprovedAt: approval.ApprovedAt,
		},
		authorized: true,
	}, nil
}

// Sender sends approved messages.
type Sender interface {
	Send(ctx context.Context, msg ApprovedMessage) error
}

// DisabledSender refuses to deliver messages.
type DisabledSender struct{}

func (DisabledSender) Send(_ context.Context, msg ApprovedMessage) error {
	if err := requireApproved(msg); err != nil {
		return err
	}
	return fmt.Errorf("sending is disabled: %w", ErrSendingDisabled)
}

// Recorder records approved messages in memory.
type Recorder struct {
	mu       sync.Mutex
	messages []Message
}

func (r *Recorder) Send(_ context.Context, msg ApprovedMessage) error {
	if err := requireApproved(msg); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, msg.Message())
	return nil
}

// Messages returns a defensive copy of the recorded messages.
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()

	messages := make([]Message, len(r.messages))
	copy(messages, r.messages)
	return messages
}

func requireApproved(msg ApprovedMessage) error {
	if !msg.Authorized() {
		return fmt.Errorf("cannot send an unapproved message: %w", ErrNotApproved)
	}
	return nil
}
