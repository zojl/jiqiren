package reply

import (
	"testing"

	"github.com/SevereCloud/vksdk/v2/object"
)

func TestIsPrivate(t *testing.T) {
	cases := []struct {
		name   string
		peerId int
		fromId int
		want   bool
	}{
		{"private chat", 123, 123, true},
		{"group chat", 2000000001, 123, false},
		{"community/other peer", 456, 123, false},
		{"zero peer", 0, 0, false},
	}

	for _, c := range cases {
		params := MakeParams(&object.MessagesMessage{PeerID: c.peerId, FromID: c.fromId}, nil)
		if got := params.isPrivate(); (got != c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
