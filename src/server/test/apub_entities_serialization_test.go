package test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"rss_parrot/dto"
	"testing"
)

func Test_Deserialize_Note(t *testing.T) {
	var bytes []byte
	var err error
	var note dto.Note

	// Note from GoToSocial: 'tag' is an object; 'cc' missing; 'to' is a string
	bytes, err = fs.ReadFile("data/note-01.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &note)
	assert.Nil(t, err)
	assert.Zero(t, len(note.Cc))
	assert.Equal(t, 1, len(note.To))
	assert.Equal(t, "https://rss-parrot.zydeo.net/u/birb", note.To[0])
	assert.NotNil(t, note.Tag)
	assert.Equal(t, 1, len(*note.Tag))
	assert.Equal(t, "https://rss-parrot.zydeo.net/u/birb", (*note.Tag)[0].Href)
	assert.Equal(t, "@birb@rss-parrot.zydeo.net", (*note.Tag)[0].Name)
	assert.Equal(t, "Mention", (*note.Tag)[0].Type)

	// Note from Mastodon: 'tag' is array; 'cc' and 'to' are arrays
	bytes, err = fs.ReadFile("data/note-02.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &note)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(note.Cc))
	assert.Equal(t, "https://toot.community/users/gaborparrot/followers", note.Cc[0])
	assert.Equal(t, "https://rss-parrot.zydeo.net/u/birb", note.Cc[1])
	assert.Equal(t, 1, len(note.To))
	assert.Equal(t, "https://www.w3.org/ns/activitystreams#Public", note.To[0])
	assert.NotNil(t, note.Tag)
	assert.Equal(t, 1, len(*note.Tag))
	assert.Equal(t, "https://rss-parrot.zydeo.net/u/birb", (*note.Tag)[0].Href)
	assert.Equal(t, "@birb@rss-parrot.zydeo.net", (*note.Tag)[0].Name)
	assert.Equal(t, "Mention", (*note.Tag)[0].Type)
}
