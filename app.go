package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aetaric/go-plex-client"
	"github.com/hugolgst/rich-go/client"
)

// App struct
type App struct {
	ctx        context.Context
	plex       plex.Plex
	pin        string
	status     string
	authToken  string
	authorized bool
	servers    []plex.PMSDevices
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
	err := client.Login("DISCORD_APP_ID")
	if err != nil {
		panic(err)
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
	//go a.VerifyCode(info.ID, p.ClientIdentifier)

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
	fmt.Print("Authorized.")
	a.authorized = true
	a.authToken = authToken

	// Get list of servers from plex
	server_plex, err := plex.New("", a.authToken)

	if err != nil {
		panic("Auth token went bad")
	}

	servers, err := server_plex.GetServers()

	if err != nil {
		panic("failed getting plex servers")
	}

	a.servers = servers
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
	for _, server := range a.servers {
		servers = append(servers, server.Name)
	}
	return servers
}

// Pick Server
func (a *App) SetServer(server int) {
	chosen_server := a.servers[server]
	plex, err := plex.New(chosen_server.PublicAddress, a.authToken)

	if err != nil {
		panic("unable to connect to plex server")
	}
	a.plex = *plex
	go a.Listener()
}

// Listener loop
func (a *App) Listener() {
	ctrlC := make(chan os.Signal, 1)
	onError := func(err error) {
		fmt.Println(err)
	}
	// loop forever. We'll wait on the server connection to appear before we do things
	for {
		events := plex.NewNotificationEvents()
		events.OnPlaying(func(n plex.NotificationContainer) {
			mediaID := n.PlaySessionStateNotification[0].RatingKey
			sessionID := n.PlaySessionStateNotification[0].SessionKey
			var title string

			sessions, err := a.plex.GetSessions()

			if err != nil {
				fmt.Printf("failed to fetch sessions on plex server: %v\n", err)
				return
			}

			for _, session := range sessions.MediaContainer.Metadata {
				if sessionID != session.SessionKey {
					continue
				} else {
					break
				}
			}

			metadata, err := a.plex.GetMetadata(mediaID)

			if err != nil {
				fmt.Printf("failed to get metadata for key %s: %v\n", mediaID, err)
			} else {
				title = metadata.MediaContainer.Metadata[0].Title
			}

			act := client.Activity{Details: fmt.Sprintf("Playing: %s", title)}
			client.SetActivity(act)
		})

		a.plex.SubscribeToNotifications(events, ctrlC, onError)
	}
}
