package main

import (
	"embed"

	"github.com/alecthomas/kong"
	cmd "github.com/religiosa1/auth_server/internal/cmd"
)

//go:embed static
var staticFiles embed.FS

type CLI struct {
	User    UserCmd   `cmd:"" help:"Manage users"`
	Session SessCmd   `cmd:"" help:"Manage sessions"`
	Serve   cmd.Serve `cmd:"" default:"withargs" help:"Run HTTP server"`
}

type UserCmd struct {
	Add    cmd.UserAdd    `cmd:"" help:"Add a new user"`
	Delete cmd.UserDelete `cmd:"" help:"Delete a user"`
	Rename cmd.UserRename `cmd:"" help:"Rename a user"`
	List   cmd.UserList   `cmd:"" default:"withargs" help:"List all users"`
}

type SessCmd struct {
	Add    cmd.SessionAdd    `cmd:"" help:"Create a session (login)"`
	Delete cmd.SessionDelete `cmd:"" help:"Delete sessions"`
	List   cmd.SessionList   `cmd:"" default:"withargs" help:"List all sessions"`
}

func main() {
	cmd.StaticFiles = staticFiles

	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("auth_server"),
		kong.Description("Authentication server"),
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
