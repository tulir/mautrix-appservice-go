package appservice

import (
	"maunium.net/go/gomatrix"
	"fmt"
)

type IntentAPI struct {
	*gomatrix.Client
	bot       *gomatrix.Client
	as        *AppService
	Localpart string
	UserID    string
}

func (as *AppService) NewIntentAPI(localpart string) *IntentAPI {
	userID := fmt.Sprintf("@%s:%s", localpart, as.HomeserverDomain)
	bot := as.BotClient()
	if userID == bot.UserID {
		bot = nil
	}
	return &IntentAPI{
		Client:    as.Client(userID),
		bot:       bot,
		as:        as,
		Localpart: localpart,
		UserID:    userID,
	}
}

func (intent *IntentAPI) Register() error {
	_, _, err := intent.Client.Register(&gomatrix.ReqRegister{
		Username: intent.Localpart,
	})
	if err != nil {
		return err
	}
	return nil
}

func (intent *IntentAPI) EnsureRegistered() error {
	if intent.as.StateStore.IsRegistered(intent.UserID) {
		return nil
	}

	err := intent.Register()
	httpErr, ok := err.(gomatrix.HTTPError)
	if !ok || httpErr.RespError.ErrCode != "M_USER_IN_USE" {
		return err
	}
	intent.as.StateStore.MarkRegistered(intent.UserID)
	return nil
}

func (intent *IntentAPI) EnsureJoined(roomID string) error {
	if intent.as.StateStore.IsInRoom(intent.UserID, roomID) {
		return nil
	}

	intent.EnsureRegistered()
	resp, err := intent.JoinRoom(roomID, "", nil)
	if err != nil {
		httpErr, ok := err.(gomatrix.HTTPError)
		if !ok || httpErr.RespError.ErrCode != "M_FORBIDDEN" || intent.bot == nil {
			_, inviteErr := intent.bot.InviteUser(roomID, &gomatrix.ReqInviteUser{
				UserID: intent.UserID,
			})
			if inviteErr != nil {
				return err
			}
			resp, err = intent.JoinRoom(roomID, "", nil)
			if err != nil {
				return err
			}
		}
	}
	intent.as.StateStore.SetMembership(intent.UserID, resp.RoomID, "join")
	return nil
}

func (intent *IntentAPI) SetDisplayName(displayName string) error {
	intent.EnsureRegistered()
	return intent.Client.SetDisplayName(displayName)
}

func (intent *IntentAPI) SetAvatarURL(avatarURL string) error {
	intent.EnsureRegistered()
	return intent.Client.SetAvatarURL(avatarURL)
}
