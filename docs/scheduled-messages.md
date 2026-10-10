# Scheduled messages

Human users can schedule a single message to a public channel or their own DM.
Messages are posted under the author's account, using the normal message-created
event for unread counts, notifications, search, WebSocket updates, and bot events.

The server stores reservations in MariaDB and checks due reservations immediately
on startup and every five seconds thereafter (up to 100 per pass). Reservations
overdue during downtime are delivered after recovery. Transient database errors
leave a reservation pending for another attempt. An inactive author, lost posting
or upload permission, an unavailable or archived channel, or a missing attachment
marks the reservation failed; it remains in the author's list for cancellation.

The list contains pending and failed reservations only, with a combined limit of
100 per user. Only the author can list or cancel them. Cancellation and delivery
lock the same reservation row; a successful cancellation prevents delivery, while
cancellation after delivery returns HTTP 409. Competing server workers create
only one message. Message insertion, attachment publication, and the sent state
commit in one transaction. As with ordinary messages, in-process events are
published after commit; this does not provide a durable event outbox.

Reserved uploads have an author-only file ACL and no channel association, so
neither the binary nor its metadata appears in a channel's file list before
delivery. Successful delivery replaces the ACL with the public-channel or DM
recipient ACL and associates the files with that channel. Each upload belongs to
one reservation, preventing early publication through another reservation.
The attachment ID is retained if its file is deleted, so the worker marks the
reservation failed instead of sending a message with a missing attachment.
Cancellation deletes its uploads. A client restoring a draft must download the
files before canceling and retain the original input text.

Use the endpoints documented in [`v3-api.yaml`](./v3-api.yaml). Submit a future
RFC 3339 timestamp with an explicit time zone. Existing file size and message
length limits apply. Recurring schedules and direct edits of reservations are
not supported; cancel and create a replacement reservation instead.

Migration 44 creates `scheduled_messages` and `scheduled_message_files`. Sent
and canceled records retain identifiers and attachment associations to prevent
reuse of a private upload; their message and draft text are cleared. A matching
frontend implements the clock button, reservation list, and draft restoration.
