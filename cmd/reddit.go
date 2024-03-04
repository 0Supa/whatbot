package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0supa/degen-go/client"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"

	discordClient "github.com/0supa/degen-go/client/discord"
)

type RedditMediaMetadata struct {
	Type     string `json:"e"`
	MimeType string `json:"m"`
	ID       string `json:"id"`
}

type RedditVideo struct {
	BitrateKbps       int    `json:"bitrate_kbps"`
	DashURL           string `json:"dash_url"`
	Duration          int    `json:"duration"`
	FallbackURL       string `json:"fallback_url"`
	HasAudio          bool   `json:"has_audio"`
	Height            int    `json:"height"`
	HlsURL            string `json:"hls_url"`
	IsGif             bool   `json:"is_gif"`
	ScrubberMediaURL  string `json:"scrubber_media_url"`
	TranscodingStatus string `json:"transcoding_status"`
	Width             int    `json:"width"`
}

type RedditMedia struct {
	RedditVideo RedditVideo `json:"reddit_video"`
}

type RedditPostData struct {
	ID            string                         `json:"id"`
	Subreddit     string                         `json:"subreddit"`
	Title         string                         `json:"title"`
	Selftext      string                         `json:"selftext"`
	Author        string                         `json:"author"`
	MediaMetadata map[string]RedditMediaMetadata `json:"media_metadata"`
	Media         RedditMedia                    `json:"media"`
	Created       float64                        `json:"created"`
	Score         int                            `json:"score"`
	Comments      int                            `json:"num_comments"`
	DirectURL     string                         `json:"url_overridden_by_dest"`
	Gallery       bool                           `json:"is_gallery"`
	NSFW          bool                           `json:"over_18"`
	Stickied      bool                           `json:"stickied"`
}

type RedditPost struct {
	Kind string         `json:"kind"`
	Data RedditPostData `json:"data"`
}

type RedditData struct {
	After   string       `json:"after"`
	Dist    int          `json:"dist"`
	ModHash string       `json:"modhash"`
	Posts   []RedditPost `json:"children"`
}

type RedditResponse struct {
	Kind    string     `json:"kind"`
	Data    RedditData `json:"data"`
	Message string     `json:"message"`
	Error   int        `json:"error"`
}

type RedditCacheEntry struct {
	expiry   time.Time
	response RedditResponse
}

var redditCache = map[string]RedditCacheEntry{}

var repeatedPosts = map[string]map[string]bool{}

var postLimit = 40

func init() {
	RegisterCommand(Command{
		Name: "reddit",
		DiscordData: api.CreateCommandData{
			Name:        "reddit",
			Description: "Sends a random post from your chosen subreddit's front page",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "subreddit",
					Description: "The subreddit's name you want to get a random post from",
					Required:    true,
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			pChannel := cmd.Event.Channel

			if cmd.Event.Channel.ParentID != 0 {
				parent, err := discordClient.Handler.Channel(cmd.Event.Channel.ParentID)
				if err == nil && parent.Type == discord.GuildText {
					pChannel = parent
				}
			}

			var options struct {
				Subreddit string `discord:"subreddit"`
			}
			if err := cmd.Data.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			subreddit := strings.TrimPrefix(strings.ToLower(options.Subreddit), "r/")

			chCacheKey := fmt.Sprintf("%s:%s", subreddit, pChannel.ID)

			if repeatedPosts[chCacheKey] == nil {
				repeatedPosts[chCacheKey] = make(map[string]bool)
			}

			entry := redditCache[subreddit]
			if time.Now().After(entry.expiry) {
				req, _ := http.NewRequest("GET", "https://y.supa.sh/?u="+
					url.QueryEscape(fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%v", subreddit, postLimit)), nil)
				req.Header.Set("User-Agent", client.GetFakeUA())

				res, err := client.HTTP.Do(req)
				if err != nil {
					return ErrorResponse(err)
				}

				if res.StatusCode != http.StatusOK {
					if res.StatusCode == http.StatusNotFound {
						return Response("Subreddit not found, please input only the subreddit name")
					}
					return ErrorResponse(errors.New("Failed getting subreddit data: " + res.Status))
				}

				reddit := RedditResponse{}
				if err := json.NewDecoder(res.Body).Decode(&reddit); err != nil {
					return ErrorResponse(err)
				}

				if reddit.Error != 0 {
					return ErrorResponse(fmt.Errorf("%v", reddit))
				}

				entry.expiry = time.Now().Add(30 * time.Minute)
				entry.response = reddit

				if subreddit != "random" {
					redditCache[subreddit] = entry
				}
			}

			filteredResponse := RedditResponse{}
			for _, post := range entry.response.Data.Posts {
				if !post.Data.Stickied &&
					!repeatedPosts[chCacheKey][post.Data.ID] &&
					(pChannel.NSFW || !post.Data.NSFW) {
					filteredResponse.Data.Posts = append(filteredResponse.Data.Posts, post)
				}
			}

			res := filteredResponse

			if len(res.Data.Posts) == 0 {
				if len(repeatedPosts[chCacheKey]) == 0 {
					return Response("Subreddit has no eligible posts")
				}

				repeatedPosts[chCacheKey] = make(map[string]bool)
				return Response("♻ Front page posts have all been posted! If you try again, you will receive repeated results")
			}

			post := res.Data.Posts[rand.Intn(len(res.Data.Posts))]
			repeatedPosts[chCacheKey][post.Data.ID] = true

			postBody := post.Data.Selftext

			embeds := []discord.Embed{}

			var galleryBody strings.Builder
			if len(post.Data.MediaMetadata) != 0 {
				for _, m := range post.Data.MediaMetadata {
					ext := "jpg"
					if m.MimeType != "" {
						ext = strings.SplitN(m.MimeType, "/", 2)[1]
					}
					mediaURL := fmt.Sprintf("https://i.redd.it/%s.%s", m.ID, ext)

					if post.Data.Gallery {
						if len(embeds) < 10 {
							embeds = append(embeds, discord.Embed{
								Title: fmt.Sprintf("Gallery (%d images)", len(post.Data.MediaMetadata)),
								URL:   post.Data.DirectURL,
								Image: &discord.EmbedImage{
									URL: mediaURL,
								},
							})
						}
						continue
					}

					galleryBody.WriteString(mediaURL + "\n")
				}
			} else if videoURL := post.Data.Media.RedditVideo.FallbackURL; videoURL != "" {
				galleryBody.WriteString(strings.TrimSuffix(videoURL, "?source=fallback") + "\n")
			} else if post.Data.DirectURL != "" {
				galleryBody.WriteString(post.Data.DirectURL + "\n")
			}

			return &api.InteractionResponseData{
				AllowedMentions: &api.AllowedMentions{},
				Content: option.NewNullableString(
					fmt.Sprintf(
						`
> [__r/%s__](<https://www.reddit.com/r/%s/>) • [u/%s](<https://www.reddit.com/u/%s/>)
> %s <t:%d:R>
### [%s](<https://reddit.com/%s>)
%s%s
`,
						post.Data.Subreddit, post.Data.Subreddit,
						post.Data.Author, post.Data.Author,
						fmt.Sprintf("`👍%v 💬%v`", post.Data.Score, post.Data.Comments), int(post.Data.Created),
						html.UnescapeString(post.Data.Title), post.Data.ID,
						html.UnescapeString(galleryBody.String()), strings.ReplaceAll(html.UnescapeString(postBody), `\\`, ``)),
				),
				Embeds: &embeds,
			}
		},
	})
}
