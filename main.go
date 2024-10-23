package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type UserSession struct {
	Session  *discordgo.Session
	Username string
}

type Bot struct {
	UserSessions []UserSession
	ChannelID    string
}

func NewBot() *Bot {
	return &Bot{}
}

func (b *Bot) Initialize() error {
	tokens, err := readFile("tokens.txt")
	if err != nil {
		return fmt.Errorf("failed to read tokens: %v", err)
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no tokens found in tokens.txt")
	}

	b.UserSessions = make([]UserSession, 0, len(tokens))
	for _, token := range tokens {
		session, err := discordgo.New(token)
		if err != nil {
			log.Printf("Error creating Discord session: %v", err)
			continue
		}

		if err := session.Open(); err != nil {
			log.Printf("Error opening connection: %v", err)
			session.Close()
			continue
		}

		username, err := b.fetchUsername(session)
		if err != nil {
			log.Printf("Error fetching username: %v", err)
			session.Close()
			continue
		}

		b.UserSessions = append(b.UserSessions, UserSession{
			Session:  session,
			Username: username,
		})
	}

	return nil
}

func (b *Bot) Close() {
	for _, us := range b.UserSessions {
		us.Session.Close()
	}
}

func (b *Bot) fetchUsername(s *discordgo.Session) (string, error) {
	user, err := s.User("@me")
	if err != nil {
		return "", err
	}
	fmt.Printf("Logged in as: %s\n", user.Username)
	return user.Username, nil
}

func (b *Bot) sendMessage(session *discordgo.Session, channelID, content string) error {
	_, err := session.ChannelMessageSend(channelID, content)
	return err
}

func (b *Bot) StartMessageLoop(delay time.Duration, shouldLoop bool) error {
	messages, err := readFile("msg.txt")
	if err != nil {
		return fmt.Errorf("failed to read messages: %v", err)
	}

	currentSession := 0
	for {
		for _, msg := range messages {
			us := b.UserSessions[currentSession]
			if err := b.sendMessage(us.Session, b.ChannelID, msg); err != nil {
				log.Printf("Error sending message with user %s: %v", us.Username, err)
				continue
			}
			fmt.Printf("message '%s' sent successfully by %s wait %d seconds for next message\n",
				msg, us.Username, int(delay.Seconds()))
			time.Sleep(delay)
		}

		currentSession = (currentSession + 1) % len(b.UserSessions)
		if !shouldLoop && currentSession == 0 {
			break
		}
	}
	return nil
}

func readFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func getInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func main() {
	fmt.Println("Welcome to discord management by sora")

	bot := NewBot()
	if err := bot.Initialize(); err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	defer bot.Close()

	for {
		fmt.Println("\nChoose option:")
		fmt.Println("1. Send message to channel")
		fmt.Println("2. Exit")

		switch getInput("Your choice: ") {
		case "1":
			bot.ChannelID = getInput("Input channel ID: ")

			delayStr := getInput("Input delay (sec): ")
			delay, err := strconv.Atoi(delayStr)
			if err != nil {
				log.Printf("Invalid delay input: %v", err)
				continue
			}

			shouldLoop := strings.ToLower(getInput("loop ? (y/n): ")) == "y"

			if err := bot.StartMessageLoop(time.Duration(delay)*time.Second, shouldLoop); err != nil {
				log.Printf("Error in message loop: %v", err)
			}

		case "2":
			fmt.Println("Exiting program. Goodbye!")
			return

		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}
