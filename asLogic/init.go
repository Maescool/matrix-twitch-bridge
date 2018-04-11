package asLogic

import (
	"github.com/Nordgedanken/matrix-twitch-bridge/asLogic/room"
	"github.com/Nordgedanken/matrix-twitch-bridge/asLogic/twitch"
	"github.com/Nordgedanken/matrix-twitch-bridge/asLogic/user"
	"github.com/Nordgedanken/matrix-twitch-bridge/asLogic/util"
	"github.com/fatih/color"
	"github.com/matrix-org/gomatrix"
	"log"
	"maunium.net/go/mautrix-appservice-go"
	"strings"
)

func Init() {
	var boldGreen = color.New(color.FgGreen).Add(color.Bold)
	appservice.GenerateRegistration("twitch", "twitch", true, true)
	boldGreen.Println("Please restart the Appservice with \"--config\"-flag applied")
}

var realUsers map[string]*user.RealUser

func Run(cfgFile string) error {
	var err error
	util.Config, err = appservice.Load(cfgFile)
	if err != nil {
		return err
	}

	queryHandler := QueryHandler{}
	//TODO Make sure to load them from a DB!!!!
	queryHandler.twitchRooms = make(map[string]string)
	queryHandler.twitchUsers = make(map[string]*user.ASUser)
	realUsers = make(map[string]*user.RealUser)

	util.Config.Init(queryHandler)

	util.Config.Listen()

	// INIT ROOM BRIDGES
	//TOKEN NEEDS TO BE A BOT
	//USERNAME NEEDS TO BE A BOT
	//twitch.Connect(token, username)
	//twitch.Listen(q.twitchUsers, q.twitchRooms)

	for {
		select {
		case event := <-util.Config.Events:
			switch event.Type {
			case "m.room.message":
				mxUser := realUsers[event.SenderID]
				if mxUser == nil {
					mxUser = &user.RealUser{}
					mxUser.Mxid = event.SenderID
					// Implement Auth logic and Queue the message for later!
					continue
				}
				if mxUser.TwitchWS == nil {
					if mxUser.TwitchToken != "" && mxUser.TwitchName != "" {
						mxUser.TwitchWS, err = twitch.Connect(mxUser.TwitchToken, mxUser.TwitchName)
						if err != nil {
							log.Println("[ERROR]: ", err)
							continue
						}
					}
				}

			}
		}
	}
	return nil
}

type QueryHandler struct {
	users       map[string]*user.ASUser
	aliases     map[string]*room.Room
	twitchUsers map[string]*user.ASUser
	twitchRooms map[string]string
}

func (q QueryHandler) QueryAlias(alias string) bool {
	if q.aliases[alias] != nil {
		return true
	}
	return false
}

type registerAuth struct {
	Type string `json:"type"`
}

func (q QueryHandler) QueryUser(userID string) bool {
	if q.users[userID] != nil {
		return true
	}
	asUser := user.ASUser{}
	asUser.Mxid = userID
	client, err := gomatrix.NewClient(util.Config.HomeserverURL, userID, util.Config.Registration.AppToken)
	if err != nil {
		util.Config.Log.Errorln(err)
		return false
	}
	asUser.MXClient = client
	username := strings.Split(strings.TrimPrefix(userID, "@"), ":")[0]

	registerReq := gomatrix.ReqRegister{
		Username: username,
		Auth: registerAuth{
			Type: "m.login.application_service",
		},
	}
	register, inter, err := asUser.MXClient.Register(&registerReq)
	if err != nil {
		util.Config.Log.Errorln(err)
		return false
	}
	if inter != nil || register == nil {
		util.Config.Log.Errorln("Error encountered during user registration")
		return false
	}
	client.AppServiceUserID = userID

	q.users[userID] = &asUser
	// TODO Link username to user on twitch (do some magic check if the user exists by crawling the channel page?) https://api.twitch.tv/kraken/users?login=<username>  DOC: https://dev.twitch.tv/docs/v5/
	return true
}
