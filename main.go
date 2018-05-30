package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"log"
)

type config struct{
	Prefix string `json:"prefix"`
	TokenDB string `json:"token"`
	database *sql.DB
}

var(
	configuration config
)

func main() {
	
	var database, err = sql.Open("mysql", "discord:truePass@tcp(localhost:3306)/discord")
	if err != nil{log.Print(err.Error())}
	configuration.database = database

	var res, QueryError = database.Query("SELECT prefix, token FROM settings")
	if QueryError!=nil{log.Print(QueryError.Error())}


	for res.Next(){
		res.Scan(&configuration.Prefix, &configuration.TokenDB)
	}

	bot, err := discordgo.New("Bot " + configuration.TokenDB)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	//Добавляем хук на событие
	bot.AddHandler(messageCreate)

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
	if message.Author.ID == session.State.User.ID{
		return
	}

	root := new(CommandData)
	root.LoadData(session, message)
}
