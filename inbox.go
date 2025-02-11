package goinsta

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Inbox is the direct message inbox.
//
// Inbox contains Conversations. Each conversation has InboxItems.
// InboxItems are the message of the chat.
type Inbox struct {
	insta *Instagram
	err   error

	Conversations []*Conversation `json:"threads"`

	HasNewer            bool   `json:"has_newer"` // TODO
	HasOlder            bool   `json:"has_older"`
	Cursor              string `json:"oldest_cursor"`
	UnseenCount         int    `json:"unseen_count"`
	UnseenCountTs       int64  `json:"unseen_count_ts"`
	MostRecentInviter   User   `json:"most_recent_inviter"`
	BlendedInboxEnabled bool   `json:"blended_inbox_enabled"`
	NextCursor          struct {
		CursorV2ID         float64 `json:"cursor_thread_v2_id"`
		CursorTimestampSec float64 `json:"cursor_timestamp_seconds"`
	} `json:"next_cursor"`
	PrevCursor struct {
		CursorV2ID         float64 `json:"cursor_thread_v2_id"`
		CursorTimestampSec float64 `json:"cursor_timestamp_seconds"`
	} `json:"prev_cursor"`
	// this fields are copied from response
	SeqID                 int64 `json:"seq_id"`
	PendingRequestsTotal  int   `json:"pending_requests_total"`
	HasPendingTopRequests bool  `json:"has_pending_top_requests"`
	SnapshotAtMs          int64 `json:"snapshot_at_ms"`
}

// Conversation is the representation of an instagram already established conversation through direct messages.
type Conversation struct {
	insta    *Instagram
	err      error
	firstRun bool

	ID   string `json:"thread_id"`
	V2ID string `json:"thread_v2_id"`
	// Items can be of many types.
	Items                      []*InboxItem          `json:"items"`
	Title                      string                `json:"thread_title"`
	Users                      []*User               `json:"users"`
	LeftUsers                  []*User               `json:"left_users"`
	AdminUserIDs               []int64               `json:"admin_user_ids"`
	ApprovalRequiredNewMembers bool                  `json:"approval_required_for_new_members"`
	Pending                    bool                  `json:"pending"`
	PendingScore               int64                 `json:"pending_score"`
	ReshareReceiveCount        int                   `json:"reshare_receive_count"`
	ReshareSendCount           int                   `json:"reshare_send_count"`
	ViewerID                   int64                 `json:"viewer_id"`
	ValuedRequest              bool                  `json:"valued_request"`
	LastActivityAt             int64                 `json:"last_activity_at"`
	Named                      bool                  `json:"named"`
	Muted                      bool                  `json:"muted"`
	Spam                       bool                  `json:"spam"`
	ShhModeEnabled             bool                  `json:"shh_mode_enabled"`
	ShhReplayEnabled           bool                  `json:"shh_replay_enabled"`
	IsPin                      bool                  `json:"is_pin"`
	IsGroup                    bool                  `json:"is_group"`
	IsVerifiedThread           bool                  `json:"is_verified_thread"`
	IsCloseFriendThread        bool                  `json:"is_close_friend_thread"`
	ThreadType                 string                `json:"thread_type"`
	ExpiringMediaSendCount     int                   `json:"expiring_media_send_count"`
	ExpiringMediaReceiveCount  int                   `json:"expiring_media_receive_count"`
	Inviter                    User                  `json:"inviter"`
	HasOlder                   bool                  `json:"has_older"`
	HasNewer                   bool                  `json:"has_newer"`
	HasRestrictedUser          bool                  `json:"has_restricted_user"`
	Archived                   bool                  `json:"archived"`
	LastSeenAt                 map[string]lastSeenAt `json:"last_seen_at"`
	NewestCursor               string                `json:"newest_cursor"`
	OldestCursor               string                `json:"oldest_cursor"`

	LastPermanentItem Item `json:"last_permanent_item"`
}

// InboxItem is any conversation message.
type InboxItem struct {
	ID            string `json:"item_id"`
	UserID        int64  `json:"user_id"`
	Timestamp     int64  `json:"timestamp"`
	ClientContext string `json:"client_context"`
	IsShhMode     bool   `json:"is_shh_mode"`
	TqSeqID       int    `json:"tq_seq_id"`

	// Type there are a few types:
	// text, like, raven_media, action_log, media_share, reel_share
	Type string `json:"item_type"`

	// Text is message text.
	Text string `json:"text"`

	// InboxItemLike is the heart that your girlfriend send to you.
	// (or in my case: the heart that my fans sends to me hehe)

	Like string `json:"like"`

	Reel      *reelShare `json:"reel_share"`
	Media     *Item      `json:"media_share"`
	ActionLog *actionLog `json:"action_log"`
}

type inboxResp struct {
	Inbox                 Inbox  `json:"inbox"`
	MostRecentInviter     *User  `json:"most_recent_inviter"`
	SeqID                 int64  `json:"seq_id"`
	PendingRequestsTotal  int    `json:"pending_requests_total"`
	SnapshotAtMs          int64  `json:"snapshot_at_ms"`
	Status                string `json:"status"`
	HasPendingTopRequests bool   `json:"has_pending_top_requests"`
}

type threadResp struct {
	Conversation *Conversation `json:"thread"`
	Status       string        `json:"status"`
}

type msgResp struct {
	Action  string `json:"action"`
	Payload struct {
		ClientContext string `json:"client_context"`
		ItemID        string `json:"item_id"`
		ThreadID      string `json:"thread_id"`
		Timestamp     string `json:"timestamp"`
	} `json:"payload"`
	Status     string `json:"status"`
	StatusCode string `json:"status_code"`
}

type reelShare struct {
	Text        string `json:"text"`
	IsPersisted bool   `json:"is_reel_persisted"`
	OwnderID    int64  `json:"reel_owner_id"`
	Type        string `json:"type"`
	ReelType    string `json:"reel_type"`
	Media       Item   `json:"media"`
}

type actionLog struct {
	Description string `json:"description"`
}

func newInbox(insta *Instagram) *Inbox {
	seqID, _ := strconv.ParseInt(randNum(6), 10, 64)
	return &Inbox{insta: insta, SeqID: seqID}
}

type lastSeenAt struct {
	Timestamp string `json:"timestamp"`
	ItemID    string `json:"item_id"`
}

func (inbox *Inbox) sync(pending bool, params map[string]string) error {
	endpoint := urlInbox
	if pending {
		endpoint = urlInboxPending
	}

	insta := inbox.insta
	body, _, err := insta.sendRequest(
		&reqOptions{
			Endpoint: endpoint,
			Query:    params,
		},
	)
	if err != nil {
		return err
	}

	resp := &inboxResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return err
	}

	inbox.updateState(resp)
	return nil
}

func (inbox *Inbox) next(pending bool, params map[string]string) bool {
	endpoint := urlInbox
	if pending {
		endpoint = urlInboxPending
	}
	if inbox.err != nil {
		return false
	}
	insta := inbox.insta
	body, _, err := insta.sendRequest(
		&reqOptions{
			Endpoint: endpoint,
			Query:    params,
		},
	)
	if err != nil {
		inbox.err = err
		return false
	}

	resp := &inboxResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		inbox.err = err
		return false
	}

	inbox.updateState(resp)

	if inbox.Cursor == "" || !inbox.HasOlder {
		inbox.err = ErrNoMore
		return false
	}
	return true
}

// Sync updates inbox messages.
func (inbox *Inbox) Sync() error {
	return inbox.sync(false, map[string]string{
		"visual_message_return_type": "unseen",
		"persistentBadging":          "true",
		"limit":                      "0",
	})
}

// SyncPending updates inbox pending messages.
func (inbox *Inbox) SyncPending() error {
	return inbox.sync(true, map[string]string{})
}

// New will send a message to a user in an existring message thread if it exists,
//   if not, it will create a new one. It will return the Conversation object,
//   for further messages you can call Conversation.Send()
//
func (inbox *Inbox) New(user *User, text string) (*Conversation, error) {
	insta := inbox.insta

	// Get existing conversation, or create a new one
	conv, err := inbox.getUserThread(user)
	if err != nil {
		return nil, err
	}
	if conv.ID != "0" {
		return conv, conv.Send(text)
	}

	to, err := prepareRecipients(user.ID)
	if err != nil {
		return nil, err
	}

	clientContext := "68" + randNum(17)
	query := map[string]string{
		"recipient_users":      to,
		"action":               "send_item",
		"is_shh_mode":          "0",
		"send_attribution":     "message_button",
		"client_context":       clientContext,
		"text":                 text,
		"device_id":            insta.dID,
		"mutation_token":       clientContext,
		"_uuid":                insta.uuid,
		"offline_threading_id": clientContext,
	}
	err = conv.send(query)
	if err != nil {
		return nil, err
	}

	err = conv.Refresh()
	if err != nil {
		return nil, err
	}
	return conv, nil
}

func (c *Conversation) send(query map[string]string) error {
	body, _, err := c.insta.sendRequest(
		&reqOptions{
			Endpoint: urlInboxSend,
			IsPost:   true,
			Query:    query,
		},
	)
	if err != nil {
		return err
	}

	var resp msgResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	c.ID = resp.Payload.ThreadID

	ts, _ := strconv.ParseInt(resp.Payload.Timestamp, 10, 64)
	msg := &InboxItem{
		ID:            resp.Payload.ItemID,
		ClientContext: resp.Payload.ClientContext,
		Timestamp:     ts,
		Type:          "text",
	}
	c.addMessage(msg)
	return nil
}

// Reset sets inbox cursor at the beginning.
func (inbox *Inbox) Reset() {
	inbox.Cursor = ""
}

// Next allows pagination over message threads.
func (inbox *Inbox) Next() bool {
	return inbox.next(false, map[string]string{
		"persistentBadging": "true",
		"cursor":            inbox.Cursor,
	})
}

// InitialSnapshot fetches the initial messages on app open, and is called
//   from Instagram.OpenApp() automatically.
func (inbox *Inbox) InitialSnapshot() bool {
	return inbox.next(false, map[string]string{
		"visual_message_return_type": "unseen",
		"thread_message_limit":       "10",
		"persistentBadging":          "true",
		"limit":                      "20",
		"fetch_reason":               "initial_snapshot",
	})
}

// NextPending allows pagination over pending messages.
func (inbox *Inbox) NextPending() bool {
	return inbox.next(true, map[string]string{
		"cursor": inbox.Cursor,
	})
}

func (inbox *Inbox) getUserThread(user *User) (*Conversation, error) {
	for _, c := range inbox.Conversations {
		if c.ThreadType == "private" && c.Users[0].ID == user.ID {
			return c, nil
		}
	}

	insta := inbox.insta
	body, _, err := insta.sendRequest(
		&reqOptions{
			Endpoint: urlGetByParticipants,
			Query: map[string]string{
				"recipient_users": fmt.Sprintf("[%d]", user.ID),
				"seq_id":          toString(inbox.SeqID + 1),
				"limit":           "20",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Thread *Conversation `json:"thread"`
		Status string        `json:"status"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Thread != nil {
		resp.Thread.setValues(insta)
		return resp.Thread, nil
	}
	return &Conversation{insta: insta, ID: "0"}, nil
}

// Error will return Conversation.err
func (c *Conversation) Error() error {
	return c.err
}

// Error will return Inbox.err
func (inbox *Inbox) Error() error {
	return inbox.err
}

func (c Conversation) lastItemID() string {
	n := len(c.Items)
	if n == 0 {
		return ""
	}
	return c.Items[n-1].ID
}

// DEPRICATED - doesn't work anymore
// Like sends heart to the conversation
//
// See example: examples/media/likeAll.go
// func (c *Conversation) Like() error {
// 	insta := c.insta
// 	to, err := prepareRecipients(c)
// 	if err != nil {
// 		return err
// 	}
//
// 	thread, err := json.Marshal([]string{c.ID})
// 	if err != nil {
// 		return err
// 	}
//
// 	data := insta.prepareDataQuery(
// 		map[string]interface{}{
// 			"recipient_users": to,
// 			"client_context":  generateUUID(),
// 			"thread_ids":      string(thread),
// 			"action":          "send_item",
// 		},
// 	)
// 	_, _, err = insta.sendRequest(
// 		&reqOptions{
// 			Endpoint: urlInboxSendLike,
// 			Query:    data,
// 			IsPost:   true,
// 		},
// 	)
// 	return err
// }

// Send sends message in conversation
func (c *Conversation) Send(text string) error {
	insta := c.insta
	// I DON'T KNOW WHY BUT INSTAGRAM WANTS A DOUBLE SLICE OF INTS FOR ONE ID. << lol
	to, err := prepareRecipients(c)
	if err != nil {
		return err
	}

	// I DONT KNOW WHY BUT INSTAGRAM WANTS SLICE OF STRINGS FOR ONE ID. << lol
	thread, err := json.Marshal([]string{c.ID})
	if err != nil {
		return err
	}
	query := map[string]string{
		"recipient_users": to,
		"client_context":  generateUUID(),
		"thread_ids":      string(thread),
		"action":          "send_item",
		"text":            text,
		"_uuid":           insta.uuid,
		"device_id":       insta.dID,
	}

	err = c.send(query)
	return err
}

// Write is like Send but being compatible with io.Writer.
func (c *Conversation) Write(b []byte) (int, error) {
	n := len(b)
	return n, c.Send(string(b))
}

// Next loads older messages if available. If not it will call Refresh().
func (c *Conversation) Next() bool {
	if c.err != nil {
		return false
	}

	cursor := c.lastItemID()
	if cursor == "" {
		err := c.Refresh()
		if err != nil {
			c.err = err
			return false
		}
		return true
	}

	err := c.callThread(
		map[string]string{
			"cursor":    cursor,
			"direction": "older",
		},
	)
	if err != nil {
		c.err = err
		return false
	}
	return true
}

// Refresh will fetch a conversation's unseen messages.
func (c *Conversation) Refresh() error {
	return c.callThread()
}

// MarkAsSeen will marks a message as seen.
func (c *Conversation) MarkAsSeen(msg InboxItem) error {
	insta := c.insta
	token := "68" + randNum(17)
	body, _, err := insta.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(urlInboxMsgSeen, c.ID, msg.ID),
			IsPost:   true,
			Query: map[string]string{
				"thread_id":            c.ID,
				"action":               "mark_seen",
				"client_context":       token,
				"_uuid":                insta.uuid,
				"offline_threading_id": token,
			},
		},
	)
	if err != nil {
		return err
	}
	var resp struct {
		Status string `json:"status"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("Status not ok while calling msg seen, '%s'", resp.Status)
	}
	return nil
}

func (c *Conversation) callThread(extras ...map[string]string) error {
	insta := c.insta
	query := map[string]string{
		"visual_message_return_type": "unseen",
		"seq_id":                     toString(c.insta.Inbox.SeqID + 1),
		"limit":                      "20",
	}
	for _, extra := range extras {
		query = MergeMapS(query, extra)
	}

	body, _, err := insta.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(urlInboxThread, c.ID),
			Query:    query,
		},
	)
	if err != nil {
		return err
	}

	resp := threadResp{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Conversation != nil {
		c.update(resp.Conversation)
	}
	if !c.HasOlder {
		return err
	}
	return nil
}

func (inbox *Inbox) updateState(resp *inboxResp) {
	insta := inbox.insta
	oldConv := inbox.Conversations

	*inbox = resp.Inbox
	inbox.insta = insta
	if resp.MostRecentInviter != nil {
		inbox.MostRecentInviter = *resp.MostRecentInviter
		inbox.MostRecentInviter.insta = insta
	}
	inbox.SeqID = resp.SeqID
	inbox.PendingRequestsTotal = resp.PendingRequestsTotal
	inbox.HasPendingTopRequests = resp.HasPendingTopRequests
	inbox.SnapshotAtMs = resp.SnapshotAtMs
	inbox.Conversations = oldConv
	for _, conv := range resp.Inbox.Conversations {
		inbox.updateConv(conv)
	}
}

func (inbox *Inbox) updateConv(c *Conversation) {
	insta := inbox.insta
	c.setValues(insta)

	for _, old := range inbox.Conversations {
		if old.ID == c.ID {
			old.update(c)
			return
		}
	}
	inbox.Conversations = append([]*Conversation{c}, inbox.Conversations...)
}

func (c *Conversation) update(newConv *Conversation) {
	insta := c.insta
	oldItems := c.Items
	newConv.setValues(insta)

	*c = *newConv
	c.Items = oldItems

	for _, msg := range newConv.Items {
		c.addMessage(msg)
	}
}

func (c *Conversation) addMessage(msg *InboxItem) {
	msg.setValues(c.insta)
	for _, m := range c.Items {
		if msg.ID == m.ID {
			*m = *msg
			return
		}
	}
	if len(c.Items) == 0 {
		c.Items = []*InboxItem{msg}
		return
	} else if msg.Timestamp > c.Items[0].Timestamp {
		// If newer than newest
		c.Items = append([]*InboxItem{msg}, c.Items...)
		return
	} else if msg.Timestamp < c.Items[len(c.Items)-1].Timestamp {
		// if older than oldest
		c.Items = append(c.Items, msg)
		return
	}
	// if somewhere in between
	for i, m := range c.Items {
		if msg.Timestamp > m.Timestamp {
			l := append([]*InboxItem{msg}, c.Items[i:]...)
			c.Items = append(c.Items[:i], l...)
		}
	}
}

func (c *Conversation) setValues(insta *Instagram) {
	c.insta = insta

	for _, msg := range c.Items {
		msg.setValues(insta)
	}

	c.Inviter.insta = insta
	for _, u := range c.Users {
		u.insta = insta
	}
	for _, u := range c.LeftUsers {
		u.insta = insta
	}
}

func (msg *InboxItem) setValues(insta *Instagram) {
	if msg.Reel != nil {
		msg.Reel.Media.insta = insta
		msg.Reel.Media.User.insta = insta
	}
	if msg.Media != nil {
		msg.Media.insta = insta
		msg.Media.User.insta = insta
	}
}
