package goinsta

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"time"
)

// Timeline is the object to represent the main feed on instagram, the first page that shows the latest feeds of my following contacts.
type Timeline struct {
	insta       *Instagram
	err         error
	errChan     chan error
	lastRequest int64
	pullRefresh bool
	sessionID   string
	prevReason  string
	fetchExtra  bool

	endpoint string
	Items    []*Item
	Tray     *Tray

	MoreAvailable         bool
	NextID                string
	NumResults            float64
	PreloadDistance       float64
	PullToRefreshWindowMs float64
	RequestID             string
	SessionID             string
}

type feedCache struct {
	Items []struct {
		Media_or_ad *Item `json:"media_or_ad"`
		EndOfFeed   struct {
			Pause    bool   `json:"pause"`
			Title    string `json:"title"`
			Subtitle string `json:"subtitle"`
		} `json:"end_of_feed_demarcator"`
	} `json:"feed_items"`

	MoreAvailable               bool    `json:"more_available"`
	NextID                      string  `json:"next_max_id"`
	NumResults                  float64 `json:"num_results"`
	PullToRefreshWindowMs       float64 `json:"pull_to_refresh_window_ms"`
	RequestID                   string  `json:"request_id"`
	SessionID                   string  `json:"session_id"`
	ViewStateVersion            string  `json:"view_state_version"`
	AutoLoadMore                bool    `json:"auto_load_more_enabled"`
	IsDirectV2Enabled           bool    `json:"is_direct_v2_enabled"`
	ClientFeedChangelistApplied bool    `json:"client_feed_changelist_applied"`
	PreloadDistance             float64 `json:"preload_distance"`
	Status                      string  `json:"status"`
	FeedPillText                string  `json:"feed_pill_text"`
	StartupPrefetchConfigs      struct {
		Explore struct {
			ContainerModule          string `json:"containermodule"`
			ShouldPrefetch           bool   `json:"should_prefetch"`
			ShouldPrefetchThumbnails bool   `json:"should_prefetch_thumbnails"`
		} `json:"explore"`
	} `json:"startup_prefetch_configs"`
	UseAggressiveFirstTailLoad bool    `json:"use_aggressive_first_tail_load"`
	HideLikeAndViewCounts      float64 `json:"hide_like_and_view_counts"`
}

func newTimeline(insta *Instagram) *Timeline {
	time := &Timeline{
		insta:    insta,
		endpoint: urlTimeline,
		errChan:  make(chan error, 1),
	}
	return time
}

// Next allows pagination after calling:
// User.Feed
// returns false when list reach the end.
// if Timeline.Error() is ErrNoMore no problem have been occurred.
// starts first request will be a cold start
func (tl *Timeline) Next(p ...interface{}) bool {
	if tl.err != nil {
		return false
	}

	insta := tl.insta
	endpoint := tl.endpoint

	// make sure at least 4 sec after last request, at most 6 sec
	var th int64 = 4
	var thR float64 = 2

	// if fetching extra, no big timeout is needed
	if tl.fetchExtra {
		th = 2
		thR = 1
	}

	if delta := time.Now().Unix() - tl.lastRequest; delta < th {
		s := time.Duration(rand.Float64()*thR + float64(th-delta))
		time.Sleep(s * time.Second)
	}
	t := time.Now().Unix()

	var reason string
	isPullToRefresh := "0"
	query := map[string]string{
		"feed_view_info":      "[]",
		"timezone_offset":     timeOffset,
		"device_id":           insta.uuid,
		"request_id":          generateUUID(),
		"_uuid":               insta.uuid,
		"bloks_versioning_id": bloksVerID,
	}

	var tWarm int64 = 10
	if tl.pullRefresh || (!tl.MoreAvailable && t-tl.lastRequest < tWarm*60) {
		reason = "pull_to_refresh"
		isPullToRefresh = "1"
		tl.sessionID = generateUUID()
		go tl.fetchTray("pull_to_refresh")
	} else if tl.lastRequest == 0 || (tl.fetchExtra && tl.prevReason == "warm_start_fetch") {
		reason = "cold_start_fetch"
		tl.sessionID = generateUUID()
		go tl.fetchTray("cold_start")
	} else if t-tl.lastRequest > tWarm*60 { // 10 min
		reason = "warm_start_fetch"
		tl.sessionID = generateUUID()
		go tl.fetchTray("warm_start_with_feed")
	} else if tl.fetchExtra || tl.MoreAvailable && tl.NextID != "" {
		reason = "pagination"
		query["max_id"] = tl.NextID
	}

	query["reason"] = reason
	query["is_pull_to_refresh"] = isPullToRefresh
	query["session_id"] = tl.sessionID
	tl.prevReason = reason

	body, _, err := insta.sendRequest(
		&reqOptions{
			Endpoint: endpoint,
			IsPost:   true,
			Gzip:     true,
			Query:    query,
			ExtraHeaders: map[string]string{
				"X-Ads-Opt-Out":  "0",
				"X-Google-AD-ID": insta.adid,
				"X-Fb":           "1",
			},
		},
	)
	if err == nil {
		tl.lastRequest = t

		// Decode json
		tmp := feedCache{}
		d := json.NewDecoder(bytes.NewReader(body))
		d.UseNumber()
		err = d.Decode(&tmp)

		// Add posts to Timeline object
		if err == nil {
			// copy constants over
			tl.NextID = tmp.NextID
			tl.MoreAvailable = tmp.MoreAvailable
			if tl.fetchExtra {
				tl.NumResults += tmp.NumResults
			} else {
				tl.NumResults = tmp.NumResults
			}
			tl.PreloadDistance = tmp.PreloadDistance
			tl.PullToRefreshWindowMs = tmp.PullToRefreshWindowMs
			tl.fetchExtra = false

			// copy post items over
			for _, i := range tmp.Items {
				// will be nil if end of feed, EndOfFeed will then be set
				if i.Media_or_ad != nil {
					setToItem(i.Media_or_ad, tl)
					tl.Items = append(tl.Items, i.Media_or_ad)
				}
			}

			// Set index value
			for i, v := range tl.Items {
				v.Index = i
			}

			if !tl.MoreAvailable {
				err = ErrNoMore
			}

			// fetch more posts if not enough posts were returned, mimick apk behvaior
			if tmp.NumResults < tmp.PreloadDistance && tmp.MoreAvailable {
				tl.fetchExtra = true
				tl.Next()
			}

			// Check if stories returned an error
			select {
			case err := <-tl.errChan:
				if err != nil {
					tl.err = err
					return false
				}
			default:
			}

			return tl.MoreAvailable
		} else {
			tl.err = err
		}
	} else {
		tl.err = err
	}

	return false
}

// SetPullRefresh will set a flag to refresh the timeline on subsequent .Next() call
func (tl *Timeline) SetPullRefresh() {
	tl.pullRefresh = true
}

// UnsetPullRefresh will unset the pull to refresh flag, if you previously manually
//   set it, and want to unset it.
func (tl *Timeline) UnsetPullRefresh() {
	tl.pullRefresh = false
}

// ClearPosts will unreference the current list of post items. Used when calling
//   .Refresh()
func (tl *Timeline) ClearPosts() {
	tl.Items = []*Item{}
	tl.Tray = &Tray{}
}

func (tl *Timeline) fetchTray(reason string) {
	body, _, err := tl.insta.sendRequest(
		&reqOptions{
			Endpoint: urlStories,
			IsPost:   true,
			Query: map[string]string{
				"supported_capabilities_new": `[{"name":"SUPPORTED_SDK_VERSIONS","value":"100.0,101.0,102.0,103.0,104.0,105.0,106.0,107.0,108.0,109.0,110.0,111.0,112.0,113.0,114.0,115.0,116.0,117.0"},{"name":"FACE_TRACKER_VERSION","value":"14"},{"name":"segmentation","value":"segmentation_enabled"},{"name":"COMPRESSION","value":"ETC2_COMPRESSION"},{"name":"world_tracker","value":"world_tracker_enabled"},{"name":"gyroscope","value":"gyroscope_enabled"}]`,
				"reason":                     reason,
				"timezone_offset":            timeOffset,
				"tray_session_id":            generateUUID(),
				"request_id":                 generateUUID(),
				"_uuid":                      tl.insta.uuid,
			},
		},
	)
	if err != nil {
		tl.errChan <- err
		return
	}

	tray := &Tray{}
	err = json.Unmarshal(body, tray)
	if err != nil {
		tl.errChan <- err
		return
	}

	tray.set(tl.insta)
	tl.Tray = tray
	tl.errChan <- nil
}

// Refresh will clear the current list of posts, perform a pull to refresh action,
//   and refresh the current timeline.
func (tl *Timeline) Refresh() error {
	tl.ClearPosts()
	tl.SetPullRefresh()
	if !tl.Next() {
		return tl.err
	}
	return nil
}

// NewFeedPostsExist will return true if new feed posts are available.
func (tl *Timeline) NewFeedPostsExist() (bool, error) {
	insta := tl.insta

	body, err := insta.sendSimpleRequest(urlFeedNewPostsExist)
	if err != nil {
		return false, err
	}

	var resp struct {
		NewPosts bool   `json:"new_feed_posts_exist"`
		Status   string `json:"status"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return false, err
	}
	return resp.NewPosts, nil
}

// Stories is a helper function to get the stories
func (tl *Timeline) Stories() []*Reel {
	return tl.Tray.Stories
}

// helper function to get the Broadcasts
func (tl *Timeline) Broadcasts() []*Broadcast {
	return tl.Tray.Broadcasts
}

func (tl *Timeline) GetNextID() string {
	return tl.NextID
}

// Delete is only a placeholder, it does nothing
func (tl *Timeline) Delete() error {
	return nil
}

func (tl *Timeline) getInsta() *Instagram {
	return tl.insta
}

// Error will the error of the Timeline instance if one occured
func (tl *Timeline) Error() error {
	return tl.err
}
