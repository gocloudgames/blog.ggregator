package main

import (
	"blog/internal/config"
	"blog/internal/database"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("wrong login parameters")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err != nil {
		return err
	}

	s.cfg.CurrentUserName = cmd.args[0]
	if err := s.cfg.SetUser(); err != nil {
		return err
	}

	fmt.Println("user" + s.cfg.CurrentUserName + "has been set")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("wrong login parameters")
	}

	u, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	})

	if err != nil {
		return err
	}

	s.cfg.CurrentUserName = u.Name
	s.cfg.SetUser()
	fmt.Println(u, "has been registered")
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())

	if err != nil {
		return err
	}

	fmt.Println("all users has been deleted")
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetAllFeeds(context.Background())

	if err != nil {
		return err
	}

	for _, feed := range feeds {
		userName, err := s.db.GetUserNameById(context.Background(), feed.UserID)
		if err != nil {
			fmt.Println(err)
			continue
		}

		result := fmt.Sprintf("%s\n%s\n%s", feed.Name, feed.Url, userName)
		fmt.Println(result)
	}

	return nil
}

func handlerFollow(s *state, cmd command, u database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("wrong follow parameters")
	}

	//It takes a single url argument and creates a new feed follow record for the current user.
	url := cmd.args[0]
	userName := s.cfg.CurrentUserName

	feed, err := s.db.GetFeedNameByUrl(context.Background(), url)
	if err != nil {
		fmt.Println(2)
		return err
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    u.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		fmt.Println(3)
		return err
	}

	fmt.Println(userName)
	fmt.Println(feed.Name)

	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetAllUsers(context.Background())

	if err != nil {
		return err
	}

	for _, user := range users {
		result := fmt.Sprintf("* %s", user.Name)
		if user.Name == s.cfg.CurrentUserName {
			result += " (current)"
		}
		fmt.Println(result)
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("invalid parrameters")
	}

	time_between_reqs := cmd.args[0]

	dur, err := time.ParseDuration(time_between_reqs)
	if err != nil {
		return err
	}

	fmt.Println("Collecting feeds every " + dur.String())

	ticker := time.NewTicker(dur)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddfeed(s *state, cmd command, u database.User) error {
	if len(cmd.args) != 2 {
		return errors.New("invalid parameters")
	}

	name := cmd.args[0]
	url := cmd.args[1]

	fmt.Println("get " + name + "from url:" + url)

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    u.ID,
	})

	if err != nil {
		return err
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    u.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		fmt.Println(3)
		return err
	}

	fmt.Println(feed)
	return nil
}

func handlerFollowing(s *state, cmd command, u database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), u.ID)

	if err != nil {
		return err
	}

	fmt.Println(u.Name)
	for _, follow := range follows {
		fmt.Println(follow.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, u database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("invalid parameters")
	}

	url := cmd.args[0]

	err := s.db.DeleteFeedFollowByUserAndFeedURL(context.Background(),
		database.DeleteFeedFollowByUserAndFeedURLParams{
			UserID: u.ID,
			Url:    url,
		})

	if err != nil {
		return err
	}

	return nil
}

type commands struct {
	cMap map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	f, exists := c.cMap[cmd.name]
	if !exists {
		return errors.New("unsupport command:" + cmd.name)
	}
	return f(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cMap[name] = f
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	// Send HTTP GET request
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set custom User-Agent
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	unescaped := html.UnescapeString(string(data))
	unescaped = html.UnescapeString(unescaped)

	var rss RSSFeed
	if err := xml.Unmarshal([]byte(unescaped), &rss); err != nil {
		return nil, err
	}

	return &rss, nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, c command) error {
		u, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)

		if err != nil {
			return err
		}

		return handler(s, c, u)
	}
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return err
	}

	rss, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	for _, item := range rss.Channel.Item {
		fmt.Println(item.Title)
	}
	return nil
}
