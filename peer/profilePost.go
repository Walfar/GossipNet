package peer

import (
	"go.dedis.ch/cs438/types"
	"time"
)

type TablePostProfile map[types.PostProfile]bool

func InitiatePosts() TablePostProfile {
	t := make(TablePostProfile)
	return t
}

func (t TablePostProfile) AddPost(msg string) error {
	postProfile := types.PostProfile{
		Message: msg,
		Date:    time.Now(),
	}

	t[postProfile] = true
	return nil
}

func (t TablePostProfile) GetPosts() TablePostProfile {
	// remove old value
	for post := range t {
		if post.Date.Add(time.Hour * 24).Before(time.Now()) {
			delete(t, post)
		}
	}
	return t
}

/*
func main() {
	fmt.Println("----------------------")
	t := InitiatePosts("me")
	t.AddPost("putain")


	time.Sleep(10 * time.Second)
	posts := t.GetPosts()
	//posts := t

	for post, _ := range posts {
		fmt.Println("-"+post.message)
	}
}

*/
