package modules

import (
	"fmt"
	"html"
	"strings"

	"github.com/AnimeKaizoku/cacher"
	"github.com/anonyindian/logger"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/parsemode/stylisehelper"
	"github.com/gigauserbot/giga/bot/helpmaker"
	"github.com/gigauserbot/giga/db"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

var afkCache = cacher.NewCacher[int64, bool](&cacher.NewCacherOpts{})

func (m *module) LoadAfk(dispatcher dispatcher.Dispatcher) {
	var l = m.Logger.Create("AFK")
	defer l.ChangeLevel(logger.LevelInfo).Println("LOADED")
	helpmaker.SetModuleHelp("afk", `
	This module provides help for the Away-From-Keyboard mode.
	
	<b>Commands</b>:
	 • <code>.afk `+html.EscapeString("<on/off> <reason>")+`</code>: Use this command to turn on/off AFK mode.   
`)
	dispatcher.AddHandler(handlers.NewCommand("afk", authorised(afk)))
	dispatcher.AddHandlerToGroup(handlers.NewMessage(filters.Message.All, checkAfk), 1)
}

func afk(ctx *ext.Context, u *ext.Update) error {
	args := strings.Fields(u.EffectiveMessage.Text)
	chat := u.EffectiveChat()
	if len(args) > 1 {
		switch args[1] {
		case "on", "true":
			reason := ""
			if len(args) > 2 {
				reason = strings.Join(args[2:], " ")
			}
			go db.UpdateAFK(true, reason)
			ctx.EditMessage(chat.GetID(), &tg.MessagesEditMessageRequest{
				ID: u.EffectiveMessage.ID,
				Message: fmt.Sprintf("Turned on AFK mode.%s", func() string {
					if reason != "" {
						return fmt.Sprintf("\nReason: %s", reason)
					}
					return reason
				}()),
			})
		case "off", "false":
			go db.UpdateAFK(false, "")
			ctx.EditMessage(chat.GetID(), &tg.MessagesEditMessageRequest{
				ID:      u.EffectiveMessage.ID,
				Message: "Turned off AFK mode.",
			})
		default:
			ctx.EditMessage(chat.GetID(), &tg.MessagesEditMessageRequest{
				ID:      u.EffectiveMessage.ID,
				Message: "AFK: Invalid Arguments",
			})
		}
	} else {
		ctx.EditMessage(chat.GetID(), &tg.MessagesEditMessageRequest{
			ID:      u.EffectiveMessage.ID,
			Message: "AFK: No arguments were provided.",
		})
	}
	return dispatcher.EndGroups
}

func checkAfk(ctx *ext.Context, u *ext.Update) error {
	chat := u.EffectiveChat()
	user := u.EffectiveUser()
	if u.EffectiveMessage.Out {
		return nil
	}
	if user != nil && user.Bot {
		// Don't reply to bots ffs
		return nil
	}
	if !(u.EffectiveMessage.Mentioned || (chat.IsAUser() && chat.GetID() != ctx.Self.ID)) {
		return nil
	}
	if _, ok := afkCache.Get(user.ID); ok {
		return nil
	}
	afkCache.Set(user.ID, true)
	afk := db.GetAFK()
	if !afk.Toggle {
		return nil
	}
	text := stylisehelper.Start(styling.Plain("I'm currently AFK"))
	if afk.Reason != "" {
		text.Plain("\nReason: ").Code(afk.Reason)
	}
	ctx.Reply(u, ext.ReplyTextStyledTextArray(text.StoArray), nil)
	return nil
}
