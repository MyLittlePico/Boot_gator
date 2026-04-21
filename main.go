package main

import (
	"database/sql"
	"fmt"
	"os"
	"github.com/MyLittlePico/Boot_Blog_Aggregator/internal/config"
	"github.com/MyLittlePico/Boot_Blog_Aggregator/internal/database"
	_ "github.com/lib/pq"
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
	c.register("users", handlerGetUsers)
	c.register("agg", handlerAgg)
	c.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	c.register("feeds", handlerGetFeedsInfo)
	c.register("follow",  middlewareLoggedIn(handlerFollow))
	c.register("following",  middlewareLoggedIn(handlerFollowing))
	c.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	c.register("browse", middlewareLoggedIn(handlerBrowse))
}

