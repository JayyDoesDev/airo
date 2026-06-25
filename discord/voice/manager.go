package voice

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	mu          sync.Mutex
	connections = map[string]*discordgo.VoiceConnection{}
)

func Join(s *discordgo.Session, guildID, channelID string) (*discordgo.VoiceConnection, error) {
	mu.Lock()
	defer mu.Unlock()

	if vc, ok := connections[guildID]; ok && vc.Ready {
		return vc, nil
	}

	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, err
	}
	connections[guildID] = vc
	return vc, nil
}

func Leave(guildID string) {
	mu.Lock()
	defer mu.Unlock()

	if vc, ok := connections[guildID]; ok {
		vc.Disconnect()
		delete(connections, guildID)
	}
}

func Get(guildID string) (*discordgo.VoiceConnection, bool) {
	mu.Lock()
	defer mu.Unlock()
	vc, ok := connections[guildID]
	return vc, ok && vc.Ready
}
