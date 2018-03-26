package view_state

import (
	"github.com/whereswaldon/arbor/lib/messages"
	"log"
	"sync"
)

type MessageStore interface {
	Get(uuid string) *messages.Message
	Seen(messageId string) bool
	MarkSeen(messageId string)
	Add(msg *messages.Message)
	Children(id string) []string
	Leaf(id string) string
	GetItems(leafId string, maxLength int) (items []*messages.Message, query string)
}

type ThreadView struct {
	MessageStore
	Thread    []*messages.Message
	CursorID  string
	LeafID    string
	ReplyToId string
	sync.RWMutex
}

func New(messages MessageStore) ThreadView {
	return ThreadView{
		MessageStore: messages,
		LeafID:       "",
		CursorID:     "",
	}
}

func (t *ThreadView) ReplyTo(id string) {
	t.Lock()
	t.ReplyToId = id
	t.Unlock()
}

func (t ThreadView) IsReplying() bool {
	t.RLock()
	replying := t.ReplyToId != ""
	t.RUnlock()
	return replying
}

func (t ThreadView) GetReplyId() string {
	t.RLock()
	id := t.ReplyToId
	t.RUnlock()
	return id
}

func (t *ThreadView) ClearReply() {
	t.Lock()
	t.ReplyToId = ""
	t.Unlock()
}

// UpdateLeaf sets the provided UUID as the ID of the current "leaf"
// message within the view of the conversation *if* it is a child of
// the previous current "leaf" message. If there is no cursor, the new
// leaf will be set as the cursor.
func (t *ThreadView) UpdateLeaf(id string) {
	msg := t.Get(id)
	t.Lock()
	if msg.Parent == t.LeafID || t.LeafID == "" {
		t.LeafID = msg.UUID
	}
	if t.CursorID == "" {
		t.CursorID = msg.UUID
	}
	t.Unlock()
}

// Refresh regenerates the threadview based upon the current leaf. It returns
// the id of any messages that need to be queried, if any
func (t *ThreadView) Refresh() string {
	items, query := t.GetItems(t.LeafID, 1024)
	t.Lock()
	t.Thread = items // save the computed ancestry of the current thread
	t.Unlock()
	return query
}

// Cursor returns the ID of the current cursor message
func (t *ThreadView) Cursor() string {
	t.RLock()
	defer t.RUnlock()
	return t.CursorID
}

func (t *ThreadView) MoveCursorTowardLeaf() {
	t.Lock()
	defer t.Unlock()

	prev := IndexOfMessageId(t.CursorID, t.Thread) - 1
	if prev >= 0 {
		t.CursorID = t.Thread[prev].UUID
	}
}

func (t *ThreadView) MoveCursorTowardRoot() {
	t.Lock()
	defer t.Unlock()
	msg := t.Get(t.Cursor())
	if msg == nil {
		log.Println("Error fetching cursor message: %s", t.CursorID)
	} else if msg.Parent == "" {
		log.Println("Cannot move cursor up, nil parent for message: %v", msg)
	} else if t.Get(msg.Parent) == nil {
		log.Println("Refusing to move cursor onto nonlocal message with id", msg.Parent)
	} else {
		t.CursorID = msg.Parent
	}
}

func (t *ThreadView) FindNewLeaf(cursor string) {
	t.Lock()
	defer t.Unlock()
	t.LeafID = t.Leaf(cursor)
}

func (t *ThreadView) Ancestry() []*messages.Message {
	return t.Thread
}

func IndexOfMessageId(element string, inContainer []*messages.Message) int {
	for i, e := range inContainer {
		if e.UUID == element {
			return i
		}
	}
	return -1
}
