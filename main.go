package main

import (
	"blog/internal/config"
	"blog/internal/database"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	s := state{}

	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.cfg = &cfg

	db, err := sql.Open("postgres", s.cfg.DbUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s.db = database.New(db)

	cmds := commands{}
	cmds.cMap = make(map[string]func(*state, command) error)
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddfeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))

	if len(os.Args) < 2 {
		fmt.Println("no enough arguments")
		os.Exit(1)
	}

	fmt.Println(os.Args)
	cmd := command{}
	cmd.name = os.Args[1]
	cmd.args = os.Args[2:]

	err = cmds.run(&s, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
