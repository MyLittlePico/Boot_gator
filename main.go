package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"github.com/google/uuid"
	"github.com/MyLittlePico/Blog_Aggregator/internal/config"
	"github.com/MyLittlePico/Blog_Aggregator/internal/database"
	_ "github.com/lib/pq"
	"time"
)


func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
	}
	db, err := sql.Open("postgres", cfg.DbURL)
	dbQueries := database.New(db)

	myState := state{
		db: dbQueries,
		cfg : &cfg,	
	}

	myCommands := commands{
		handlers : make(map[string]func(*state, command) error, 0),
	}

	myCommands.init()

	myArgs := os.Args
	if len(myArgs) < 2 {
		fmt.Println("no command found")
		os.Exit(1)
	}

	myCommand := command{
		name : myArgs[1],
		args : myArgs[2:],
	}

	if err:= myCommands.run(&myState, myCommand); err != nil{
		fmt.Println(err)
		os.Exit(1)
	}

}

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error{
	fun, ok := c.handlers[cmd.name]
	if !ok{
		return fmt.Errorf("Command %s not exist", cmd.name)
	}
	return fun(s, cmd)
}

func (c *commands) register(name string, f func(s *state, cmd command)error) error{
	c.handlers[name] = f
	return nil
}

func (c *commands)init(){
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid number of arguments")
	}
	_ , err :=s.db.GetUser(context.Background(),cmd.args[0])
	if err != nil{
		fmt.Printf("couldn't log in user %s not exist",cmd.args[0])
		os.Exit(1)
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
		fmt.Printf("User %s already exist",cmd.args[0])
		os.Exit(1)
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
		fmt.Printf("Restting Database Failed: %v\n",err)
		os.Exit(1)
	}
	fmt.Println("Restting Database successed")
	return nil
}
