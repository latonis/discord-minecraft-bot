package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// An MCStatus struct stores the information retrieved from mcstatus.io about a Minecraft server.
type MCStatus struct {
	Online      bool   `json:"online"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	EulaBlocked bool   `json:"eula_blocked"`
	RetrievedAt int64  `json:"retrieved_at"`
	ExpiresAt   int64  `json:"expires_at"`
	Version     struct {
		NameRaw   string `json:"name_raw"`
		NameClean string `json:"name_clean"`
		NameHTML  string `json:"name_html"`
		Protocol  int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Online int `json:"online"`
		Max    int `json:"max"`
		List   []struct {
			UUID      string `json:"uuid"`
			NameRaw   string `json:"name_raw"`
			NameClean string `json:"name_clean"`
			NameHTML  string `json:"name_html"`
		} `json:"list"`
	} `json:"players"`
	Motd struct {
		Raw   string `json:"raw"`
		Clean string `json:"clean"`
		HTML  string `json:"html"`
	} `json:"motd"`
	Icon     string `json:"icon"`
	Mods     []any  `json:"mods"`
	Software any    `json:"software"`
	Plugins  []any  `json:"plugins"`
}

// This function checks an error passed into it and prints the resulting message if such an error is found.
func check_err(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// This function forms the appropriate API query for mcstatus.io depending on if a server is on Bedrock or Java
// and returns the information retrieved from the API.
func server_status(server string, bedrock bool) MCStatus {
	fmt.Println("Getting information for server: " + server)
	platform := "java"

	if bedrock {
		platform = "bedrock"
	}

	resp, err := http.Get("https://api.mcstatus.io/v2/status/" + platform + "/" + server)
	check_err(err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	check_err(err)

	var result MCStatus
	err = json.Unmarshal(body, &result)
	check_err(err)

	return result
}

// This function registers the slash (`/command`) commands to Discord so users know which commands they can issue to the bot.
func register_commands(session *discordgo.Session) {
	guild_id := os.Getenv("DISCORD_GUILD_ID")
	app_id := os.Getenv("DISCORD_APP_ID")
	_, err := session.ApplicationCommandBulkOverwrite(app_id, guild_id, []*discordgo.ApplicationCommand{
		{
			Name:        "status",
			Description: "Display the status of players currently on the Minecraft Server",
		},
		{
			Name:        "version",
			Description: "Display the current version of the Minecraft Server",
		},
		{
			Name:        "players",
			Description: "Display the current players on the Minecraft Server",
		},
	})

	check_err(err)
}

// This function is the main driver of the bot.
func main() {
	discord_key := os.Getenv("DISCORD_KEY")
	session, err := discordgo.New("Bot " + discord_key)
	check_err(err)

	register_commands(session)

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.Content == "hello" {
			s.ChannelMessageSend(m.ChannelID, "world")
		}

	})

	status := server_status(os.Getenv("MINECRAFT_SERVER"), false)

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()
		switch data.Name {
		case "status":
			server_up := "offline"
			server_up_emoji := ":x:"
			if status.Online {
				server_up = "online"
				server_up_emoji = ":green_circle:"
			}
			err := session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "## " + status.Host + "\nServer Status: " + server_up + " " + server_up_emoji +
						"\nPlayers Online: " + strconv.Itoa(status.Players.Online) + " :tools:\nMaximum Players: " +
						strconv.Itoa(status.Players.Max) + " :chart_with_upwards_trend:",
				},
			})
			check_err(err)

		case "version":
			err := session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "## " + status.Host + "\nVersion: " + status.Version.NameClean + " :floppy_disk:",
				},
			})
			check_err(err)
		case "players":
			playerStr := "## " + status.Host + "\n" + "Players Online:\n"
			for _, player := range status.Players.List {
				playerStr = playerStr + player.NameClean + " :green_square:\n"
			}

			err := session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: playerStr,
				},
			})
			check_err(err)
		}

	})

	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = session.Open()

	check_err(err)

	err = session.UpdateGameStatus(0, "Minecraft with "+strconv.Itoa(status.Players.Online)+" TR Minecrafters")

	check_err(err)

	defer session.Close()

	fmt.Println("Discord bot spinning up")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
