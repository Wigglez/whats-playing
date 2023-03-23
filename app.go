package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/aetaric/go-plex-client"
	"github.com/aetaric/whats-playing/storage"
	"github.com/hugolgst/rich-go/client"
	"github.com/koffeinsource/go-imgur"
)

// App struct
type App struct {
	ctx           context.Context
	plex          plex.Plex
	pin           string
	status        string
	authToken     string
	server        string
	chosen_server plex.PMSDevices
	username      string
	userid        int
	authorized    bool
	imgurClient   *imgur.Client
	servers       []plex.PMSDevices
	storage       storage.Storage
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.status = "Please link your plex account to get started."
	err := client.Login("413407336082833418")
	if err != nil {
		panic(err)
	}
	imgurClient, err := imgur.NewClient(new(http.Client), "0dedf5b51d09876", "")
	if err != nil {
		fmt.Printf("failed during imgur client creation. %+v\n", err)
		return
	}
	a.imgurClient = imgurClient

	a.storage = storage.Storage{}
	a.storage.Open()

	token := a.storage.Get([]byte("plex-token"), []byte("token"))
	if token != nil {
		a.status = "Token loaded from storage"
		a.authToken = string(token)
		a.getServersFromPlex()
	}
}

func (a *App) LinkPlex() {
	// get Plex headers
	p, err := plex.New("", "abc123")
	if err != nil {
		panic(err)
	}

	// Get PIN
	info, err := plex.RequestPIN(p.Headers)

	if err != nil {
		panic("request plex pin failed: " + err.Error())
	}

	expireAtParsed, err := time.Parse(time.RFC3339, info.ExpiresAt)

	if err != nil {
		panic("could not get expiration for plex pin")
	}

	expires := time.Until(expireAtParsed).String()

	fmt.Printf("your pin %s and expires in %s\n", info.Code, expires)
	a.pin = info.Code
	a.status = fmt.Sprintf("Please Navigate to https://plex.tv/link and provide the pin: %s", info.Code)

	var authToken string
	for {
		pinInformation, _ := plex.CheckPIN(info.ID, p.ClientIdentifier)

		if pinInformation.AuthToken != "" {
			authToken = pinInformation.AuthToken
			break
		}

		time.Sleep(1 * time.Second)
	}

	a.status = "You have been successfully authorized!"
	fmt.Println("Authorized.")
	a.authToken = authToken
	a.storage.Set([]byte("plex-token"), []byte("token"), []byte(authToken))
	a.getServersFromPlex()
}

func (a *App) getServersFromPlex() {
	// Get list of servers from plex
	server_plex, err := plex.New("", a.authToken)

	if err != nil {
		panic("Auth token went bad")
	}

	user, err := server_plex.MyAccount()

	if err != nil {
		panic("failed getting user data")
	}

	a.username = user.Username
	a.userid = user.ID

	servers, err := server_plex.GetServers()

	if err != nil {
		panic("failed getting plex servers")
	}

	a.servers = servers
	a.authorized = true
}

// Display Status
func (a *App) GetStatus() string {
	return a.status
}

func (a *App) IsAuthorized() bool {
	return a.authorized
}

// Display Servers
func (a *App) GetServers() []string {
	var servers []string
	servers = append(servers, "")
	for _, server := range a.servers {
		servers = append(servers, server.Name)
	}
	return servers
}

// Pick Server
func (a *App) SetServer(server string) {
	for _, s := range a.servers {
		if s.Name == server {
			a.chosen_server = s
			a.server = server
		}
	}
	a.Listener()
}

// Listener loop
func (a *App) Listener() {

	for _, addr := range a.chosen_server.Connection {
		plex, err := plex.New(addr.URI, a.authToken)

		if err == nil {
			a.plex = *plex
		} else {
			fmt.Println("unable to connect to plex server")
		}
	}

	a.status = fmt.Sprintf("Listening for events from %s for %s", a.server, a.username)
	ctrlC := make(chan os.Signal, 1)
	onError := func(err error) {
		fmt.Println(err)
	}

	events := plex.NewNotificationEvents()
	events.OnPlaying(func(n plex.NotificationContainer) {
		mediaID := n.PlaySessionStateNotification[0].RatingKey
		sessionID := n.PlaySessionStateNotification[0].SessionKey
		viewOffset := n.PlaySessionStateNotification[0].ViewOffset
		state := n.PlaySessionStateNotification[0].State

		sessions, err := a.plex.GetSessions()

		if err != nil {
			fmt.Printf("failed to fetch sessions on plex server: %v\n", err)
			return
		}

		for _, session := range sessions.MediaContainer.Metadata {
			if sessionID != session.SessionKey {
				continue
			} else {
				fmt.Printf("user: %s, tracked_user: %v\n", session.User.Title, a.username)
				if session.User.Title == a.username {
					fmt.Printf("session: %s, Match: %s\n", sessionID, session.SessionKey)
					metadata, err := a.plex.GetMetadata(mediaID)

					if err != nil {
						fmt.Printf("failed to get metadata for key %s: %v\n", mediaID, err)
					}

					var title string
					var largeText string
					var stateText string

					switch session.Type {
					case "track":
						title = fmt.Sprintf("%s - %s", metadata.MediaContainer.Metadata[0].GrandparentTitle,
							metadata.MediaContainer.Metadata[0].Title)
						largeText = "Listening to Music"
					case "movie":
						title = fmt.Sprintf("%s (%v)", metadata.MediaContainer.Metadata[0].Title, metadata.MediaContainer.Metadata[0].Year)
						largeText = "Watching a Movie"
					case "episode":
						title = fmt.Sprintf("%s S%vE%v - %s", metadata.MediaContainer.Metadata[0].GrandparentTitle, metadata.MediaContainer.Metadata[0].ParentIndex, metadata.MediaContainer.Metadata[0].Index, metadata.MediaContainer.Metadata[0].Title)
						largeText = "Watching a TV Show"
					}

					stateText = fmt.Sprintf("%s Elapsed", time.Duration(viewOffset*1000*1000))

					var imgurerr error
					imgurURL := a.storage.Get([]byte("imgur-urls"), []byte(metadata.MediaContainer.Metadata[0].Thumb))

					if imgurURL == nil {
						thumbURL := fmt.Sprintf("%s%s?X-Plex-Token=%s", a.plex.URL, metadata.MediaContainer.Metadata[0].Thumb, a.authToken)
						resp, err := http.Get(thumbURL)
						if err != nil {
							fmt.Println("Error fetching image data from plex")
						}

						imageData, err := io.ReadAll(resp.Body)
						if err != nil {
							fmt.Println("Error reading image data from plex")
						}

						imgurData, _, imgurerr := a.imgurClient.UploadImage(imageData, "", "URL", metadata.MediaContainer.Metadata[0].Title, "")
						if imgurerr != nil {
							fmt.Println(imgurerr)
						}
						a.storage.Set([]byte("imgur-urls"), []byte(metadata.MediaContainer.Metadata[0].Thumb), []byte(imgurData.Link))
						imgurURL = []byte(imgurData.Link)
					}

					caser := cases.Title(language.AmericanEnglish)
					if imgurerr == nil {
						act := client.Activity{LargeImage: string(imgurURL), SmallImage: state, Details: title, LargeText: largeText, State: stateText, SmallText: caser.String(state)}
						client.SetActivity(act)
					} else {
						act := client.Activity{LargeImage: "logo", SmallImage: state, Details: title, LargeText: largeText, State: stateText, SmallText: caser.String(state)}
						client.SetActivity(act)
					}
				}
			}
		}
	})

	a.plex.SubscribeToNotifications(events, ctrlC, onError)
}
