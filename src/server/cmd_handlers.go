package server

import (
	"fmt"
	"net/http"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
)

type cmdbHandlerGroup struct {
	cfg         *shared.Config
	logger      shared.ILogger
	keyHandler  logic.IKeyHandler
	sender      logic.IActivitySender
	broadcaster logic.IMessenger
}

func NewCmdHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
	keyHandler logic.IKeyHandler,
	sender logic.IActivitySender,
	broadcaster logic.IMessenger,
) IHandlerGroup {
	res := cmdbHandlerGroup{
		cfg:         cfg,
		logger:      logger,
		keyHandler:  keyHandler,
		sender:      sender,
		broadcaster: broadcaster,
	}
	return &res
}

func (cmd *cmdbHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/cmd/follow", func(w http.ResponseWriter, r *http.Request) { cmd.getFollow(w, r) }},
		{"GET", "/cmd/toot", func(w http.ResponseWriter, r *http.Request) { cmd.getToot(w, r) }},
	}
}

func (cmd *cmdbHandlerGroup) getToot(w http.ResponseWriter, r *http.Request) {

	cmd.logger.Info("Toot: Request received")

	//user := cmd.cfg.Birb.User
	//cmd.broadcaster.EnqueueBroadcast(user, "2023-12-13T21:40:37Z", "Hello, world! The bird is a-tooting.")
}

func (cmd *cmdbHandlerGroup) getFollow(w http.ResponseWriter, r *http.Request) {

	cmd.logger.Info("Follow: Request received")

	activity := dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      "https://rss-parrot.zydeo.net/follow-44",
		Type:    "Follow",
		Actor:   "https://rss-parrot.zydeo.net/u/birb03",
		Object:  "https://toot.community/users/gaborparrot",
	}

	privkey, err := cmd.keyHandler.GetPrivKey(cmd.cfg.Birb.User)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, "Failed to retrieve key")
		fmt.Fprintln(w, err)
		return
	}

	err = cmd.sender.Send(privkey, cmd.cfg.Birb.User, "https://toot.community/inbox", &activity)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, "Failed to post activity")
		fmt.Fprintln(w, err)
		return
	}

	fmt.Fprintln(w, "ActivityOut posted")
}
