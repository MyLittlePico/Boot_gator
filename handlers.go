package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/MyLittlePico/Boot_gator/internal/database"
	_ "github.com/lib/pq"
	"time"
	"strings"
	"strconv"
)

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid number of arguments")
	}
	_ , err :=s.db.GetUser(context.Background(),cmd.args[0])
	if err != nil{
		return fmt.Errorf("couldn't log in user %s not exist",cmd.args[0])
	}

	if err := s.cfg.SetUser(cmd.args[0]); err != nil{
		return err
	}
	fmt.Printf("Set user name to %s\n",cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid number of arguments")
	}
	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err == nil{
		return fmt.Errorf("User %s already exist",cmd.args[0])
	}
	user, err := s.db.CreateUser(context.Background(),database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: cmd.args[0],
	})
	if err!= nil{
		return err
	}

	s.cfg.CurrentUserName = cmd.args[0]
	fmt.Printf("User %s was created", s.cfg.CurrentUserName )
	fmt.Printf("%+v\n", user)

	return handlerLogin(s , cmd)
}

func handlerReset(s *state, cmd command) error {
	err := s.db.Reset(context.Background())
	if err != nil {
		return fmt.Errorf("Restting Database Failed: %v\n",err)
	}
	fmt.Println("Restting Database successed")
	return nil
}

func handlerGetUsers(s *state, cmd command) error {
	users , err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Getting users failed: %w ",err)
	}
	for _ , user := range users{
		strToPrint := "* " + user
		if user == s.cfg.CurrentUserName{
			strToPrint += " (current)"
		}
		fmt.Println(strToPrint)
	}
	return nil
}

func handlerAgg(s *state, cmd command ) error {
	if len(cmd.args) != 1{
		return fmt.Errorf("Wrong number of arguments")
	}
	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil{
		return err
	}
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func scrapeFeeds(s *state) error{
	ctx := context.Background()
	feed, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return err
	}
	err = s.db.MarkFeedFetched(ctx, feed.ID)

	rssFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil{
		return fmt.Errorf("Fetching feed failed: %w",err)
	}

	fmt.Printf("-%s\n",rssFeed.Channel.Title)
	for _, item := range rssFeed.Channel.Item{
		layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC3339}
		
		var parsedTime time.Time
		for _, layout := range layouts{
			parsedTime, err = time.Parse(layout ,item.PubDate)
			if err == nil{
				break
			}
		}
		if err != nil {
			return fmt.Errorf("unsupport time layout")
		}
		err = s.db.CreatePost(ctx,database.CreatePostParams{
			ID			: uuid.New(),
			Title     	: item.Title,
			Url 		: item.Link,
			Description : item.Description,
			PublishedAt : parsedTime,
			FeedID   	: feed.ID,
			})
		if err != nil{
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				// ignore duplicate post
			}else {
				fmt.Printf("couldn't create post: %v", err)
			}
		} 
	}
	return nil
}


func handlerAddFeed(s *state, cmd command, user database.User) error {
	ctx := context.Background()

	if len(cmd.args) != 2 {
		return fmt.Errorf("Wrong number of arguments")
	}
	
	feed, err := s.db.CreateFeed(ctx, database.CreateFeedParams{
	
		ID        : uuid.New(),
		CreatedAt : time.Now(),
		UpdatedAt : time.Now(),
		Name      : cmd.args[0],
		Url       : cmd.args[1],
		UserID    : user.ID,
		} )
	if err != nil {
		return fmt.Errorf("Adding feed failed: %w",err)
	}

	fmt.Printf("%+v\n", feed)

	_, err = s.db.CreateFeedFollow(ctx,database.CreateFeedFollowParams{
		ID        : uuid.New(),
		CreatedAt : time.Now(),
		UpdatedAt : time.Now(),
		UserID    : user.ID,
		FeedID    : feed.ID,
		})
	return err
}

func handlerGetFeedsInfo(s *state, cmd command) error {
	feeds, err := s.db.GetFeedsInfo(context.Background())
	if err!= nil {
		return fmt.Errorf("Getting feeds failed: %w",err)
	}

	for _, feed := range(feeds) {
		fmt.Printf("-%s\n  -%s\n    -%s\n",feed.Name, feed.Url, feed.UserName.String)
	}
	
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	ctx := context.Background()
	if len(cmd.args) != 1 {
		return fmt.Errorf("Wrong number of arguments")
	}

	feed, err := s.db.GetFeed(ctx, cmd.args[0])
	if err != nil {
		return err
	}

	feedFollow, err := s.db.CreateFeedFollow(ctx,database.CreateFeedFollowParams{
		ID        : uuid.New(),
		CreatedAt : time.Now(),
		UpdatedAt : time.Now(),
		UserID    : user.ID,
		FeedID    : feed.ID,
		})
	fmt.Printf("%+v\n",feedFollow)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	feeds ,err := s.db.GetFeedFollowsForUser(context.Background(),user.ID)
	if err!=nil{
		return err
	}
	for _,feed := range feeds{
		fmt.Printf(" -%s\n",feed.FeedName)		
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state,cmd command, user database.User) error) func(*state, command) error{
	return func(s *state, cmd command) error{
		user, err := s.db.GetUser(context.Background(),s.cfg.CurrentUserName)
		if err != nil{
			return err
		}
		return handler(s, cmd, user)
	}
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	ctx := context.Background()
	
	if len(cmd.args) != 1{
		return fmt.Errorf("Wrong number of arguments")
	}
	
	feed, err := s.db.GetFeed(ctx, cmd.args[0])
	if err != nil {
		return err
	}
	
	
	err = s.db.Unfollow(ctx,database.UnfollowParams{
		UserID : user.ID,
		FeedID : feed.ID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Unfollow %s",feed.Url)
	return nil

}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2
	var err error
	if len(cmd.args) == 1{
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil{
			return err
		}
	}
	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID : user.ID,
		Limit  : int32(limit),
	})
	if err != nil{
		return err
	}
	
	for _, post := range posts {
		fmt.Printf("-%s\n",post.Title)
		fmt.Printf("  -%s\n",post.Description)

	}
	return nil
}