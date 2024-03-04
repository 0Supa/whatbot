package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

type TwitchGQLPayload struct {
	OperationName string `json:"operationName"`
	Query         string `json:"query"`
	Variables     any    `json:"variables"`
}

type TwitchGQLBaseResponse struct {
	Extensions struct {
		Duration      json.Number `json:"durationMilliseconds"`
		OperationName string      `json:"operationName"`
		RequestID     string      `json:"requestID"`
	} `json:"extensions"`
}

type TwitchUserResponse struct {
	*TwitchGQLBaseResponse
	Data struct {
		User TwitchUser `json:"user"`
	} `json:"data"`
}

type TwitchBanStatusResponse struct {
	*TwitchGQLBaseResponse
	Data struct {
		BanStatus struct {
			BannedUser TwitchUser `json:"bannedUser"`
			Moderator  TwitchUser `json:"moderator"`
			CreatedAt  string     `json:"createdAt"`
			ExpiresAt  string     `json:"expiresAt"`
			Permanent  bool       `json:"isPermanent"`
		} `json:"chatRoomBanStatus"`
	} `json:"data"`
}

type TwitchUser struct {
	ID          string `json:"id,omitempty"`
	Login       string `json:"login,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

var clientId = "ue6666qo983tsx6so1t0vnawi233wa"
var gqlToken = os.Getenv("GQL_TOKEN")

func GetTwitchUser(login string, id string) (TwitchUser, error) {
	response := TwitchUserResponse{}
	user := TwitchUser{}

	payload, err := json.Marshal(TwitchGQLPayload{
		OperationName: "User",
		Query:         "query User($login:String $id:ID) { user(login:$login id:$id) { id login displayName } }",
		Variables: TwitchUser{
			Login: login,
			ID:    id,
		},
	})
	if err != nil {
		return user, err
	}

	req, _ := http.NewRequest("POST", "https://gql.twitch.tv/gql", bytes.NewBuffer(payload))
	req.Header.Set("User-Agent", GetFakeUA())
	req.Header.Set("Client-Id", clientId)

	res, err := HTTP.Do(req)
	if err != nil {
		return user, err
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return user, err
	}

	user = response.Data.User

	return user, nil
}

func GetTwitchBan(channelID string, userID string) (TwitchBanStatusResponse, error) {
	banStatus := TwitchBanStatusResponse{}

	payload, err := json.Marshal(TwitchGQLPayload{
		OperationName: "",
		Query:         "query ($channelID:ID! $userID:ID!) { chatRoomBanStatus(channelID:$channelID userID:$userID) { bannedUser { id login } createdAt expiresAt isPermanent moderator { id login } } }",
		Variables: map[string]interface{}{
			"channelID": channelID,
			"userID":    userID,
		},
	})
	if err != nil {
		return banStatus, err
	}

	req, _ := http.NewRequest("POST", "https://gql.twitch.tv/gql", bytes.NewBuffer(payload))
	req.Header.Set("User-Agent", GetFakeUA())
	req.Header.Set("Client-Id", clientId)
	req.Header.Set("Authorization", "OAuth "+gqlToken)

	res, err := HTTP.Do(req)
	if err != nil {
		return banStatus, err
	}

	if err := json.NewDecoder(res.Body).Decode(&banStatus); err != nil {
		return banStatus, err
	}

	return banStatus, nil
}
