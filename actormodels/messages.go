package actormodels

import "github.com/AsynkronIT/protoactor-go/actor"

///////////////////////////////////////////////////////////
///////////////////////////COMMANDS////////////////////////
///////////////////////////////////////////////////////////

type Origin struct {
	GuildID   string
	ChannelID string
}

type BigEmojiMsg struct {
	Origin
	EmojiString string
}

///////////////////////////////////////////////////////////
///////////////////////////GUILD OPS///////////////////////
///////////////////////////////////////////////////////////

type GuildEnvelope struct {
	GuildID string
	Message interface{}
}

type MessageRecv struct {
	Content string
}

type GuildJoined struct{}

type GuildKicked struct{}

///////////////////////////////////////////////////////////
///////////////////////SUPERVISOR OPS//////////////////////
///////////////////////////////////////////////////////////

type QueryGuildPID struct {
	GuildID string
	// Spawn the actor if it doesn't exist in the registry
	Spawn bool
}

type QueryGuildPIDResponse struct {
	PID *actor.PID
}