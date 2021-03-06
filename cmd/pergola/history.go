package main

import (
	"fmt"
	"io"
	"log"
	"time"
	"strings"

	"github.com/jroimartin/gocui"
	wrap "github.com/mitchellh/go-wordwrap"
	vs "github.com/whereswaldon/arbor/cmd/pergola/view_state"
	"github.com/whereswaldon/arbor/lib/messages"
)

const ReplyView = "reply-view"

type History struct {
	vs.ThreadView
	ViewIDs  map[string]struct{}
	Query    chan<- string
	Outbound chan<- *messages.Message
}

// NewList creates a new History that uses the provided Tree
// to manage message history. This History acts as a layout manager
// for the gocui layout package. The method returns a History, a readonly
// channel of queries, and a readonly channel of new messages to be sent
// to the sever. The queries are message UUIDs that
// the local store has requested the message contents for.
func NewList(store *Tree) (*History, chan string, <-chan *messages.Message) {
	queryChan := make(chan string)
	outChan := make(chan *messages.Message)
	return &History{
		ThreadView: vs.New(store),
		ViewIDs:    make(map[string]struct{}),
		Query:      queryChan,
		Outbound:   outChan,
	}, queryChan, outChan
}

func (h *History) destroyOldViews(ui *gocui.Gui) {
	// destroy old views
	for id := range h.ViewIDs {
		ui.DeleteView(id)
	}
	// reset ids
	h.ViewIDs = make(map[string]struct{})

}

// Layout builds a message history in the provided UI
func (m *History) Layout(ui *gocui.Gui) error {
	m.destroyOldViews(ui)

	maxX, maxY := ui.Size()

	// get the latest history
	query := m.Refresh()
	if query != "" {
		m.Query <- query
	}
	totalY := maxY // how much vertical space is left for drawing messages

	cursorY := (totalY - 2) / 2
	cursorX := 0
	cursorId := m.Cursor()
	if cursorId == "" {
		return nil
	}
	err, cursorHeight := m.drawView(cursorX, cursorY, maxX-1, down, true, cursorId, ui) //draw the cursor message
	if err != nil {
		log.Println("error drawing cursor view: ", err)
		return err
	}

	thread := m.Ancestry()
	currentIdxBelow := vs.IndexOfMessageId(cursorId, thread)
	currentIdxAbove := currentIdxBelow

	lowerBound := cursorY + cursorHeight
	replyY := lowerBound
	for currentIdxBelow--; currentIdxBelow >= 0 && lowerBound < maxY; currentIdxBelow-- {
		err, msgHeight := m.drawView(0, lowerBound, maxX-1, down, false, thread[currentIdxBelow].UUID, ui) //draw the cursor message
		if err != nil {
			log.Println("error drawing view: ", err)
			return err
		}
		lowerBound += msgHeight
	}
	upperBound := cursorY - 1
	for currentIdxAbove++; currentIdxAbove < len(thread) && upperBound >= 0; currentIdxAbove++ {
		err, msgHeight := m.drawView(0, upperBound, maxX-1, up, false, thread[currentIdxAbove].UUID, ui) //draw the cursor message
		if err != nil {
			log.Println("error drawing view: ", err)
			return err
		}
		upperBound -= msgHeight
	}
	if m.IsReplying() {
		m.drawReplyView(0, replyY, maxX-1, 5, ui)
	}
	return nil
}

type Direction int

const up Direction = 0
const down Direction = 1

func (h *History) drawView(x, y, w int, dir Direction, isCursor bool, id string, ui *gocui.Gui) (error, int) {
	const borderHeight = 2
	const gutterWidth = 4
	msg := h.ThreadView.Get(id)
	if msg == nil {
		log.Println("accessed nil message with id:", id)
	}
	seen := h.Seen(id)
	numSiblings := len(h.Children(msg.Parent)) - 1
	contents := wrap.WrapString(msg.Content, uint(w-gutterWidth-1))
	height := strings.Count(contents, "\n") + borderHeight

	var upperLeftX, upperLeftY, lowerRightX, lowerRightY int
	if dir == up {
		upperLeftX = x + gutterWidth
		upperLeftY = y - height
		lowerRightX = x + w
		lowerRightY = y
	} else if dir == down {
		upperLeftX = x + gutterWidth
		upperLeftY = y
		lowerRightX = x + w
		lowerRightY = y + height
	}
	log.Printf("message at (%d,%d) -> (%d,%d)\n", upperLeftX, upperLeftY, lowerRightX, lowerRightY)
	if numSiblings > 0 {
		name := id + "sib"
		if v, err := ui.SetView(name, x, upperLeftY, x+gutterWidth, lowerRightY); err != nil {
			if err != gocui.ErrUnknownView {
				log.Println(err)
				return err, 0
			}
			fmt.Fprintf(v, "%d", numSiblings)
			h.ViewIDs[name] = struct{}{}
		}
	}

	if v, err := ui.SetView(id, upperLeftX, upperLeftY, lowerRightX, lowerRightY); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println(err)
			return err, 0
		}
		v.Title = id
		v.Wrap = true
		fmt.Fprint(v, contents)
		if isCursor {
			ui.SetCurrentView(id)
			h.MarkSeen(id)
			seen = true
		}
		if !seen {
			v.BgColor = gocui.ColorWhite
			v.FgColor = gocui.ColorBlack
		}
		h.ViewIDs[id] = struct{}{}

	}
	return nil, height + 1
}

func (his *History) drawReplyView(x, y, w, h int, ui *gocui.Gui) error {
	if v, err := ui.SetView(ReplyView, x, y, x+w, y+h); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println(err)
			return err
		}
		v.Title = "Reply to " + his.GetReplyId()
		v.Editable = true
		v.Wrap = true
		ui.SetKeybinding(ReplyView, gocui.KeyEnter, gocui.ModNone, his.SendReply)
	}
	ui.SetCurrentView(ReplyView)
	ui.SetViewOnTop(ReplyView)
	return nil
}

func (m *History) BeginReply(g *gocui.Gui, v *gocui.View) error {
	m.ReplyTo(m.Cursor())
	return nil
}

func (m *History) SendReply(g *gocui.Gui, v *gocui.View) error {
	if !m.IsReplying() {
		return nil
	}
	data := make([]byte, 1024)
	n, err := v.Read(data)
	if err != nil && err != io.EOF {
		log.Println("Err reading composed message", err)
		return err
	}
	id := m.GetReplyId()
	g.DeleteKeybindings(ReplyView) // apparently, deleting the view doesn't do this
	g.DeleteView(ReplyView)
	m.ClearReply()
	msg := &messages.Message{
    		Username: "pergola",
    		Timestamp: time.Now().Unix(),
		Parent:  id,
		Content: string(data[:n]),
	}
	log.Printf("Sending reply to %s: %s\n", id, string(data))
	m.Outbound <- msg
	return nil
}

func (m *History) CursorUp(g *gocui.Gui, v *gocui.View) error {
	if m.IsReplying() {
		return nil
	}

	m.MoveCursorTowardRoot()
	return nil
}

func (m *History) CursorRight(g *gocui.Gui, v *gocui.View) error {
	return m.cursorSide(right, g, v)
}

func (m *History) CursorLeft(g *gocui.Gui, v *gocui.View) error {
	return m.cursorSide(left, g, v)
}

type side int

const left side = 0
const right side = 1

func (m *History) cursorSide(s side, g *gocui.Gui, v *gocui.View) error {
	if m.IsReplying() {
		return nil
	}
	id := m.Cursor()
	msg := m.ThreadView.Get(id)
	if msg == nil {
		log.Println("Error fetching cursor message: %s", id)
		return nil
	} else if msg.Parent == "" {
		log.Println("Cannot move cursor up, nil parent for message: %v", msg)
		return nil
	} else if len(m.Children(msg.Parent)) < 2 {
		log.Println("Refusing to move cursor onto nonexistent sibling", msg.Parent)
		return nil
	} else {
		siblings := m.Children(msg.Parent)
		index := indexOf(id, siblings)
		if s == right {
			index = (index + len(siblings) + 1) % len(siblings)
		} else {
			index = (index + len(siblings) - 1) % len(siblings)
		}
		newCursor := siblings[index]
		log.Printf("Selecting new cursor (old %s) as %s from %v\n", id, newCursor, siblings)
		m.ViewSubtreeOf(newCursor)
		log.Println("Selected leaf :", m.LeafID)
		return nil
	}
}

func (m *History) CursorDown(g *gocui.Gui, v *gocui.View) error {
	if m.IsReplying() {
		return nil
	}
	m.MoveCursorTowardLeaf()
	return nil
}

func indexOf(element string, inContainer []string) int {
	for i, e := range inContainer {
		if e == element {
			return i
		}
	}
	return -1
}
