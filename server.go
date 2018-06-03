package main

import (
	"sync"

	"github.com/Strum355/go-queue/queue"
)

func (s *servers) getCount() int {
	return s.Count
}

/*
func (s *servers) validate() {
	for _, guild := range s.Server {
		details := guildDetails(guild.ID)
	}
} */

func (s *server) newVoiceInstance() {
	s.VoiceInst = &voiceInst{
		Queue:   queue.New(),
		Done:    make(chan error),
		RWMutex: new(sync.RWMutex),
	}
}

func (s server) nextSong() song {
	return s.VoiceInst.Queue.Front().(song)
}

func (s server) finishedSong() {
	s.VoiceInst.Queue.PopFront()
}

func (s server) addSong(song song) {
	s.VoiceInst.Queue.PushBack(song)
}

func (s server) queueLength() int {
	s.VoiceInst.RLock()
	defer s.VoiceInst.RUnlock()
	return s.VoiceInst.Queue.Len()
}

func (s server) iterateQueue() []song {
	s.VoiceInst.RLock()
	defer s.VoiceInst.RUnlock()
	ret := make([]song, s.VoiceInst.Queue.Len())
	for i, val := range s.VoiceInst.Queue.List() {
		ret[i] = val.(song)
	}
	return ret
}
