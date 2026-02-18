package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// DeliveryStatus values.
const (
	DeliveryPending   = 0
	DeliveryDelivered = 1
	DeliveryRead      = 2
)

// MLS message type values stored in the messages table.
const (
	MsgTypeApplication = 0
	MsgTypeCommit      = 1
	MsgTypeWelcome     = 2
	MsgTypeProposal    = 3
)

// Message represents a stored encrypted message.
type Message struct {
	ID              string
	GroupID         string
	SenderID        string
	ServerTimestamp int64
	Payload         []byte
	PayloadSize     int
	MessageType     int
	Epoch           int
	CreatedAt       int64
}

// DeliveryRecord tracks per-recipient delivery state.
type DeliveryRecord struct {
	MessageID   string
	RecipientID string
	Status      int
	DeliveredAt *int64
	ReadAt      *int64
}

// NewULID generates a new ULID.
func NewULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// InsertMessage stores a message and creates delivery_status rows for all
// group members except the sender. It returns the generated message ID.
func (s *Store) InsertMessage(ctx context.Context, groupID, senderID string, payload []byte, messageType, epoch int) (string, int64, error) {
	msgID := NewULID()
	now := time.Now()
	serverTS := now.UnixMicro()
	createdAt := now.Unix()
	payloadSize := len(payload)

	err := s.InTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO messages (id, group_id, sender_id, server_timestamp, payload, payload_size, message_type, epoch, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			msgID, groupID, senderID, serverTS, payload, payloadSize, messageType, epoch, createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert message: %w", err)
		}

		// Create delivery_status rows for all group members except sender.
		_, err = tx.ExecContext(ctx,
			`INSERT INTO delivery_status (message_id, recipient_id, status)
			 SELECT ?, user_id, 0 FROM group_members WHERE group_id = ? AND user_id != ?`,
			msgID, groupID, senderID,
		)
		if err != nil {
			return fmt.Errorf("insert delivery status: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", 0, err
	}

	return msgID, serverTS, nil
}

// GetMessagesByGroup returns messages for a group using cursor-based pagination.
// If cursor is empty, returns the most recent messages.
// direction: true = forward (newer), false = backward (older).
func (s *Store) GetMessagesByGroup(ctx context.Context, groupID, cursor string, limit int, forward bool) ([]*Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if cursor == "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, group_id, sender_id, server_timestamp, payload, payload_size, message_type, epoch, created_at
			 FROM messages WHERE group_id = ? ORDER BY id DESC LIMIT ?`,
			groupID, limit,
		)
	} else if forward {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, group_id, sender_id, server_timestamp, payload, payload_size, message_type, epoch, created_at
			 FROM messages WHERE group_id = ? AND id > ? ORDER BY id ASC LIMIT ?`,
			groupID, cursor, limit,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, group_id, sender_id, server_timestamp, payload, payload_size, message_type, epoch, created_at
			 FROM messages WHERE group_id = ? AND id < ? ORDER BY id DESC LIMIT ?`,
			groupID, cursor, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetPendingMessages returns all messages with PENDING delivery status for a user,
// ordered by server_timestamp ascending (oldest first for delivery).
func (s *Store) GetPendingMessages(ctx context.Context, recipientID string) ([]*Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.group_id, m.sender_id, m.server_timestamp, m.payload, m.payload_size, m.message_type, m.epoch, m.created_at
		 FROM delivery_status ds
		 JOIN messages m ON m.id = ds.message_id
		 WHERE ds.recipient_id = ? AND ds.status = 0
		 ORDER BY m.server_timestamp ASC`,
		recipientID,
	)
	if err != nil {
		return nil, fmt.Errorf("query pending messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// UpdateDeliveryStatus updates the delivery status for a message-recipient pair.
func (s *Store) UpdateDeliveryStatus(ctx context.Context, messageID, recipientID string, status int) error {
	var deliveredAt, readAt interface{}
	now := time.Now().UnixMicro()

	switch status {
	case DeliveryDelivered:
		deliveredAt = now
	case DeliveryRead:
		deliveredAt = now
		readAt = now
	}

	result, err := s.db.ExecContext(ctx,
		`UPDATE delivery_status SET status = ?,
		 delivered_at = COALESCE(delivered_at, ?),
		 read_at = COALESCE(read_at, ?)
		 WHERE message_id = ? AND recipient_id = ?`,
		status, deliveredAt, readAt, messageID, recipientID,
	)
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetDeliveryStatus returns the delivery record for a message-recipient pair.
func (s *Store) GetDeliveryStatus(ctx context.Context, messageID, recipientID string) (*DeliveryRecord, error) {
	d := &DeliveryRecord{}
	var deliveredAt, readAt sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT message_id, recipient_id, status, delivered_at, read_at
		 FROM delivery_status WHERE message_id = ? AND recipient_id = ?`,
		messageID, recipientID,
	).Scan(&d.MessageID, &d.RecipientID, &d.Status, &deliveredAt, &readAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get delivery status: %w", err)
	}
	if deliveredAt.Valid {
		d.DeliveredAt = &deliveredAt.Int64
	}
	if readAt.Valid {
		d.ReadAt = &readAt.Int64
	}
	return d, nil
}

// GetMessageSenderID returns the sender_id for a message. Returns ErrNotFound if
// the message does not exist.
func (s *Store) GetMessageSenderID(ctx context.Context, messageID string) (string, error) {
	var senderID string
	err := s.db.QueryRowContext(ctx,
		`SELECT sender_id FROM messages WHERE id = ?`, messageID,
	).Scan(&senderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get message sender: %w", err)
	}
	return senderID, nil
}

// DeleteExpiredMessages removes messages older than the given cutoff (Unix seconds).
// Returns the number of deleted messages.
func (s *Store) DeleteExpiredMessages(ctx context.Context, cutoffUnixSeconds int64) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM messages WHERE created_at < ?`, cutoffUnixSeconds,
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired messages: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}

func scanMessages(rows *sql.Rows) ([]*Message, error) {
	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.ID, &m.GroupID, &m.SenderID, &m.ServerTimestamp, &m.Payload,
			&m.PayloadSize, &m.MessageType, &m.Epoch, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return msgs, nil
}
