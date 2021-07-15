package websocket

// Responses

type gameListCountResponse struct {
	Live           string `json:"live"`
	Correspondence string `json:"correspondence"`
}

type pongResponse struct {
	Client int64 `json:"client"`
	Server int64 `json:"server"`
}

// {
// 	"list":"live",
// 	"by":"rank",
// 	"size":136,
// 	"where":null,
// 	"from":0,
// 	"limit":9,
// 	"results": [
// 		{
// 			"id":35307427,
// 			"group_ids":[],
// 			"phase":"play",
// 			"name":"Fast",
// 			"player_to_move":872803,
// 			"width":19,
// 			"height":19,
// 			"move_number":189,
// 			"paused":0,
// 			"private":false,
// 			"black":{
// 				"username":"eigenX",
// 				"id":872803,
// 				"rank":32.18607724253454,
// 				"professional":false,
// 				"accepted":false,
// 				"ratings":{
// 					"version":5,
// 					"overall":{
// 						"rating":2108.486295000427,
// 						"deviation":65.7108266877671,
// 						"volatility":0.05998657883830825
// 					}
// 				}
// 			},
// 			"white":{
// 				"username":"katago-micro",
// 				"id":902691,
// 				"rank":38.146915580496284,
// 				"professional":false,
// 				"accepted":false,
// 				"ratings":{
// 					"version":5,
// 					"overall":{
// 						"rating":2727.697676636648,
// 						"deviation":68.98075614493769,
// 						"volatility":0.0600185792853173
// 					}
// 				}
// 			},
// 			"time_per_move":21,
// 			"ranked":false,
// 			"handicap":10,
// 			"komi":0.5,
// 			"bot_game":true,
// 			"in_beginning":false,
// 			"in_middle":false,
// 			"in_end":true,
// 			"group_ids_map":{}
// 		},
// 	],
// }

// Requests

type pingRequest struct {
	Client  int64   `json:"client"`
	Drift   float64 `json:"drift"`
	Latency float64 `json:"latency"`
}
