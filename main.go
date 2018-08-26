package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"database/sql"
	"log"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
)

type config struct {
	Prefix   string `json:"prefix"`
	TokenDB  string `json:"token"`
	database *sql.DB
}

var (
	configuration config
	userStatus    = make(map[string]string)
	quaue         = make(map[string]int)
	reactions     = make(chan *discordgo.MessageReactionAdd)
)

func main() {

	var database, err = sql.Open("mysql", "discord:truePass@tcp(localhost:3306)/discord")
	if err != nil {
		log.Print(err.Error())
	}
	configuration.database = database

	//Load configuration
	var res, QueryError = database.Query("SELECT prefix, token FROM settings")
	if QueryError != nil {
		log.Print(QueryError.Error())
	}
	for res.Next() {
		res.Scan(&configuration.Prefix, &configuration.TokenDB)
	}
	//configuration.Prefix = "sudo "
	//Load users status
	var userSql, _ = database.Query("SELECT discord_id, status FROM users")
	for userSql.Next() {
		var tmpId, tmpStatus string
		userSql.Scan(&tmpId, &tmpStatus)
		userStatus[tmpId] = tmpStatus
	}

	//Start discordbot
	bot, err := discordgo.New("Bot " + configuration.TokenDB)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	//Добавляем хук на событие
	bot.AddHandler(messageCreate)
	bot.AddHandler(reactionAdd)

	// Open a websocket connection to Discord and begin listening.
	err = bot.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	bot.Close()
	database.Close()
}

// Перехват нового сообщения
func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {

	if len(message.Embeds) != 0 {
		var messageEmbedData = *message.Embeds[0]
		if quaue[messageEmbedData.Description] > 0 {
			for i := 0; i < quaue[messageEmbedData.Description]; i++ {
				session.MessageReactionAdd(message.ChannelID, message.ID, empojiPoll[i])
			}
			quaue[messageEmbedData.Description] = 0
		}
	}

	if message.Author.ID == session.State.User.ID {
		return
	}

	root := new(CommandData)
	root.LoadData(session, message)
}

func reactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	if elem, ok := trackReactions[r.MessageID]; ok && r.Emoji.Name == "🇫" {
		user, err := s.User(r.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		if strings.Contains(elem.Embeds[0].Fields[0].Value, user.Username) {
			return
		}
		embed := &discordgo.MessageEmbed{
			Title:  elem.Embeds[0].Title,
			Author: elem.Embeds[0].Author,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Payed respect",
					Value:  elem.Embeds[0].Fields[0].Value + "\n" + user.Username,
					Inline: false,
				},
			},
		}
		message, err := s.ChannelMessageEditEmbed(r.ChannelID, r.MessageID, embed)
		if err != nil {
			log.Println(err)
			return
		}
		trackReactions[r.MessageID] = message
	}

}
