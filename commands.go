package main

import (
	"github.com/bwmarrin/discordgo"
	"strings"
	"strconv"
	"math/rand"
	"time"
	"log"
	"fmt"
	"github.com/thanhpk/randstr"
	"github.com/fogleman/gg"
	"github.com/disintegration/imaging"
	"image/color"
	"image"
	"os"
)

type CommandData struct {
	session *discordgo.Session
	message *discordgo.MessageCreate
	author *discordgo.User
	channel string
	prefix  string
	guild *discordgo.Guild
}


var (
	empojiPoll = []string{
		"0⃣",
		"1⃣",
		"2⃣",
		"3⃣",
		"4⃣",
		"5⃣",
		"6⃣",
		"7⃣",
		"8⃣",
		"9⃣",
	}

	srcImage, _ = gg.LoadImage("spank.jpg")
)

const (
	imageSize = 400
)

func (data CommandData) LoadData(session *discordgo.Session, message *discordgo.MessageCreate){
	rand.Seed(time.Now().UTC().UnixNano())

	data.session = session
	data.message = message

	var channel, _ = data.session.State.Channel(data.message.ChannelID)
	var guild, _ = data.session.State.Guild(channel.GuildID)
	data.guild = guild
	channel, guild = nil, nil

	data.author = message.Author
	data.channel = message.ChannelID
	data.prefix = configuration.Prefix
	data.checkCommand()
}

func (data CommandData) startswith(forcheck string)(bool){
	var start = strings.Split(data.message.Content, " ")[0]
	if start ==  forcheck {
		return true
	} else {
		return false
	}
}

func (data CommandData) roll(){
	var elements = strings.Split(data.message.Content, " ")
	var first, second, result, elemLen = 0, 100, 0, len(elements)

	if elemLen == 2{
		second, _ = strconv.Atoi(elements[1])
	} else if elemLen == 3{
		first, _ = strconv.Atoi(elements[1])
		second, _ = strconv.Atoi(elements[2])
	}

	if first > second{ first, second = second, first }

	if first == second{
		result = first
	} else {
		result = rand.Intn(second-first) + first
	}

	data.session.ChannelMessageSend(data.channel, "Result: " + strconv.Itoa(result))
}

func (data CommandData) top(){
	var res, QueryError = configuration.database.Query("SELECT username, points FROM users ORDER BY points DESC LIMIT 10")

	if QueryError!=nil{log.Print(QueryError.Error())}

	var username string
	var points int
	var inline = false
	var counter = 0

	var fields []*discordgo.MessageEmbedField

	for res.Next(){
		res.Scan(&username, &points)
		counter+=1
		if counter > 1{inline=true}

		var tmp = &discordgo.MessageEmbedField{
			fmt.Sprintf("%d - %s", counter, username),
			strconv.Itoa(points),
			inline,
		}
		fields = append(fields, tmp)
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name:data.author.Username},
		Color: 0x00ff00,
		Fields: fields,
	}
	data.session.ChannelMessageSendEmbed(data.channel, embed)
}

func (data CommandData) stats(){
	var res, QueryError = configuration.database.Query("SELECT discord_id, points FROM users ORDER BY points DESC")
	if QueryError!=nil{log.Print(QueryError.Error())}

	var points int
	var id string
	var counter = 1

	for res.Next(){
		res.Scan(&id, &points)
		if id == data.author.ID{break}
		counter+=1
	}
	//log.Print("Вызван статус")
	var fields = []*discordgo.MessageEmbedField{
		{
			"Баланс",
			fmt.Sprintf("%d DGC", points),
			true,
		},
		{
			"Место",
			fmt.Sprintf("%d место среди пользователей", counter),
			true,
		},
		{
			"Статус",
			userStatus[data.author.ID],
			false,
		},
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name:data.author.Username},
		Color: 0x00ff00,
		Fields: fields,
	}
	data.session.ChannelMessageSendEmbed(data.channel, embed)
}

func (data CommandData) throw()  {
	var target = data.message.Mentions
	if len(target) == 0{return
	}
	var targetAllInfo, _ = data.session.GuildMember(data.guild.ID, target[0].ID)
	var authorAllInfo, _ = data.session.GuildMember(data.guild.ID, data.author.ID)
	var targetNick, authorNick = targetAllInfo.Nick, authorAllInfo.Nick

	if targetNick == "" { targetNick = target[0].Username }
	if authorNick == "" { authorNick = data.author.Username }

	var allEmoji = data.guild.Emojis
	var staticEmoji []*discordgo.Emoji
	for emoji := range allEmoji {
		if allEmoji[emoji].Animated == false{
			staticEmoji = append(staticEmoji, allEmoji[emoji])
		}
	}

	var targetEmoji = staticEmoji[rand.Intn(len(staticEmoji))]
	var emojiString = fmt.Sprintf("<:%s:%s>", targetEmoji.Name, targetEmoji.ID)

	var messageText = fmt.Sprintf("**%s** threw %s at **%s**", authorNick, emojiString, targetNick)

	if strings.Contains(messageText, "@everyone") || strings.Contains(messageText, "@here"){
		return
	}

	data.session.ChannelMessageSend(data.channel, messageText)
}

func checkAddInfo(data string) (*discordgo.MessageEmbedField, bool, int){
	var messageData = strings.Split(data, "\n")
	var counterAddInfo = strings.Split(messageData[0], " ")
	var questions = messageData[1:]

	if len(counterAddInfo) == 2{

		if separator, err := strconv.Atoi(counterAddInfo[1]); err == nil {
			//fmt.Printf("%q looks like a number.\n", v)
			if separator > len(questions) {
				return &discordgo.MessageEmbedField{}, false, len(questions)
			}
			var additionalInformation = &discordgo.MessageEmbedField{
				"Информация",
				strings.Join(questions[separator:], "\n"),
				false,
			}
			return additionalInformation, true, separator
		}

	}
	return &discordgo.MessageEmbedField{}, false, len(questions)
}

func (data CommandData) poll() {

	// FIRST CHECK AND INIT ZONE
	if strings.Contains(data.message.Content, "@everyone") || strings.Contains(data.message.Content, "@here") {
		return
	}
	var messageData = strings.Split(data.message.Content, "\n")
	var questions = messageData[1:]
	var fields []*discordgo.MessageEmbedField
	var variants = ""
	var pollId = randstr.RandomString(5)

	//BALANCE CHECK
	points, err := configuration.database.Query("SELECT points FROM users WHERE discord_id=?", data.message.Author.ID)
	if err != nil {
		log.Println(err.Error())
	}
	var userPoints int
	points.Next()
	points.Scan(&userPoints)
	if userPoints < 50 {
		data.session.ChannelMessageSend(data.channel, "Cначала денег накопи")
		return
	} else {
		configuration.database.Query("UPDATE users SET points = `points`-50 WHERE discord_id=?", data.message.Author.ID)
	}

	if additionalInfo, success, separator := checkAddInfo(data.message.Content); success == true {
		for i:=range questions{
			if i == separator {
				break
			}
			variants += fmt.Sprintf("%d) %s\n", i, questions[i])
		}
		if separator > 10 {
			data.session.ChannelMessageSend(data.channel, "``k3rn3l_p4n1c: questions overflow``")
			return
		}

		fields = append(fields, additionalInfo)
		fields = append(fields, &discordgo.MessageEmbedField{
			"Варианты",
			variants,
			false,
		})
		quaue[pollId] = separator
	} else {

		for i:=range questions{
			variants += fmt.Sprintf("%d) %s\n", i, questions[i])
		}
		if len(questions) > 10 {
			data.session.ChannelMessageSend(data.channel, "``k3rn3l_p4n1c: questions overflow``")
			return
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			"Варианты",
			variants,
			false,
		})
		quaue[pollId] = separator
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name:data.author.Username},
		Color: 0x00ff00,
		Fields: fields,
		Description:pollId,
	}
	data.session.ChannelMessageSendEmbed(data.channel, embed)
	data.session.ChannelMessageDelete(data.message.ChannelID, data.message.ID)
}

func (data CommandData) help(){
	var fields = []*discordgo.MessageEmbedField{
		{
			"Info",
			fmt.Sprintf("**%[1]shelp** - получить информацию о командах\n**%[1]sstats** - информация об аккаунте\n**%[1]stop** - топ пользователей\n ", data.prefix),
			true,
		},
		{
			"Misc",
			fmt.Sprintf("**%[1]sroll** - u see me rollin\n ", data.prefix),
			true,
		},
		{
			"Fun",
			fmt.Sprintf("**%[1]sspank** - :slap: :slap: :slap:\n ", data.prefix),
			true,
		},
		{
			"Tools",
			fmt.Sprintf("**%[1]spoll** - создать опрос\n ", data.prefix),
			true,
		},
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name:data.author.Username},
		Color: 0x00ff00,
		Fields: fields,
	}
	data.session.ChannelMessageSendEmbed(data.channel, embed)
}

func (data CommandData) spank(){
	var res, _ = configuration.database.Query("SELECT discord_id, points FROM users ORDER BY points DESC")

	var points int
	var id string
	var counter = 1
	for res.Next(){
		res.Scan(&id, &points)
		if id == data.author.ID{break}
		counter+=1
	}
	if counter > 30 {
		data.session.ChannelMessageSend(data.channel, fmt.Sprintf("Только топ-30 могут использовать spank. Проверить место - **%sstats**", data.prefix))
		return
	}

	var realTarget *discordgo.User
	if len(data.message.Mentions) == 0 {
		data.session.ChannelMessageSend(data.channel, "Тебе нужно выбрать кого-нибудь")
		return
	} else {
		realTarget = data.message.Mentions[0]
	}

	var tmpName = randstr.RandomString(7) + ".jpg"

	var spanked, _ = data.session.UserAvatarDecode(realTarget)
	var spanker, _ = data.session.UserAvatarDecode(data.author)
	spanker = imaging.Resize(spanker, imageSize, imageSize, imaging.Lanczos)
	spanked = imaging.Resize(spanked, imageSize, imageSize, imaging.Lanczos)

	dst := imaging.New(2000, 1333, color.NRGBA{0, 0, 0, 0})
	dst = imaging.Paste(dst, srcImage, image.Pt(0, 0))
	dst = imaging.Paste(dst, spanked, image.Pt(1470, 660))
	dst = imaging.Paste(dst, spanker, image.Pt(970, 0))
	err := imaging.Save(dst, tmpName)
	if err != nil{
		log.Println(err.Error())
	}
	file, _ := os.Open(tmpName)

	data.session.ChannelFileSend(data.channel, "spank.jpg", file)
	os.Remove(tmpName)
}

func (data CommandData) checkCommand(){
	var start = strings.Split(data.message.Content, " ")[0]
	if strings.Contains(start, "\n") {
		start = strings.Split(data.message.Content, "\n")[0]
	}
	switch start {
	case data.prefix+"roll":
		data.roll()
		break
	case data.prefix+"top":
		data.top()
		break
	case data.prefix+"stats":
		data.stats()
		break
	case data.prefix+"throw":
		data.throw()
		break
	case data.prefix+"poll":
		data.poll()
		break
	case data.prefix+"help":
		data.help()
		break
	case data.prefix+"spank":
		data.spank()
		break
	}

}