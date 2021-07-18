package websocket

// Responses

type PongResponse struct {
	Client int64 `json:"client"`
	Server int64 `json:"server"`
}

type GameListQueryResponse struct {
	List    string                 `json:"list"` // live, correspondence
	By      string                 `json:"by"`   // rank
	Size    int                    `json:"size"`
	Where   map[string]interface{} `json:"where"`
	From    int                    `json:"from"`
	Limit   int                    `json:"limit"`
	Results []*Game                `json:"results"`
}

type Game struct {
	ID           int64       `json:"id"`
	GroupIDs     interface{} `json:"group_ids"` // FIXME: Add proper type
	Phase        string      `json:"phase"`     // play
	Name         string      `json:"name"`      // fast
	PlayerToMove int64       `json:"player_to_move"`
	Width        int         `json:"width"`  // 19
	Height       int         `json:"height"` // 19
	MoveNumber   int         `json:"move_number"`
	Paused       int         `json:"paused"`
	Private      bool        `json:"private"`
	Black        *Player     `json:"black"`
	White        *Player     `json:"white"`
	TimePerMove  int         `json:"time_per_move"`
	Ranked       bool        `json:"ranked"`
	Handicap     int         `json:"handicap"`
	Komi         float32     `json:"komi"`
	BotGame      bool        `json:"bot_game"`
	InBeginning  bool        `json:"in_beginning"`
	InMiddle     bool        `json:"in_middle"`
	InEnd        bool        `json:"in_end"`
	GroupIDsMap  interface{} `json:"group_ids_map"` // FIXME: Add proper type

	// Calculated values
	MedianRating float64 `json:"-"`
}

type Player struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	Rank         float64 `json:"rank"`
	Professional bool    `json:"professional"`
	Accepted     bool    `json:"accepted"`
	Ratings      struct {
		Version int64 `json:"version"`
		Overall struct {
			Rating     float64 `json:"rating"`
			Deviation  float64 `json:"deviation"`
			Volatility float64 `json:"volatility"`
		} `json:"overall"`
	} `json:"ratings"`
}

// Requests

type GameListQueryRequest struct {
	List    string                 `json:"list"`    // live, correspondence
	SortBy  string                 `json:"sort_by"` // rank
	Where   map[string]interface{} `json:"where"`   // FIXME: Use proper type
	From    int                    `json:"from"`
	Limit   int                    `json:"limit"`
	Channel string                 `json:"channel"`
}

type PingRequest struct {
	Client  int64   `json:"client"`
	Drift   float64 `json:"drift"`
	Latency float64 `json:"latency"`
}
