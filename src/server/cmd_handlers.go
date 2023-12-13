package server

import (
	"fmt"
	"net/http"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
)

type cmdbHandlerGroup struct {
	logger shared.ILogger
	sender logic.IActivitySender
}

func NewCmdHandlerGroup(
	logger shared.ILogger,
	sender logic.IActivitySender,
) IHandlerGroup {
	res := cmdbHandlerGroup{
		logger: logger,
		sender: sender,
	}
	return &res
}

func (cmd *cmdbHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/cmd/beep", func(w http.ResponseWriter, r *http.Request) { cmd.getBeep(w, r) }},
	}
}

func (cmd *cmdbHandlerGroup) getBeep(w http.ResponseWriter, r *http.Request) {

	cmd.logger.Info("Beep: Request received")

	//activity := dto.ActivityOut{
	//	Context: "https://www.w3.org/ns/activitystreams",
	//	Id:      "https://rss-parrot.zydeo.net/users/birb/statuses/43/activity",
	//	Type:    "Create",
	//	Actor:   "https://rss-parrot.zydeo.net/users/birb",
	//	Object: dto.Note{
	//		Id:           "https://rss-parrot.zydeo.net/users/birb/statuses/43",
	//		Type:         "Note",
	//		Published:    "2023-12-10T21:15:31Z",
	//		AttributedTo: "https://rss-parrot.zydeo.net/users/birb",
	//		Content:      "<p><span class='h-card' translate='no'><a href='https://toot.community/@gaborparrot' class='u-url mention'>@<span>gaborparrot</span></a></span> Brezel boom boom</p>",
	//		To:           []string{"https://toot.community/users/gaborparrot"},
	//		Cc:           []string{},
	//	},
	//}

	activity := dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      "https://rss-parrot.zydeo.net/follow-44",
		Type:    "Follow",
		Actor:   "https://rss-parrot.zydeo.net/users/birb03",
		Object:  "https://toot.community/users/gaborparrot",
	}

	err := cmd.sender.Send("https://toot.community/inbox", &activity)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, "Failed to post activity")
		fmt.Fprintln(w, err)
	}

	fmt.Fprintln(w, "ActivityOut posted")
}
