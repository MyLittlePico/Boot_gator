package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/MyLittlePico/Blog_Aggregator/internal/database"
	_ "github.com/lib/pq"
	"time"
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

func handlerAgg(s *state, cmd command) error {
	feed, err := fetchFeed(context.Background(),"https://www.wagslane.dev/index.xml")
	if err != nil{
		return fmt.Errorf("Fetching feed failed: %w",err)
	}
	fmt.Printf("%+v\n",feed)
	return nil
}