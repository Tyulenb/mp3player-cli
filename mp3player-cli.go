package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/urfave/cli"
)

type music struct {
	Name string
	Path string
}

type playlist struct {
	Name  string
	Songs []music
}

type Status struct {
	// Used to quickly get information about playback.
	Paused        bool            // If the song paused.
	Loop          bool            // If the song looped.
	Interupted    bool            // If the song has not ended.
	Inited        bool            // Speaker should be inited only once.
	oldSmapleRate beep.SampleRate // To resample audio files.
}

func (s *Status) Reset() {
	s.Paused = false
	s.Interupted = false
}

func (s *Status) NextSongUpdate() {
	s.Loop = false
	s.Interupted = true
}

func main() {
	app := cli.NewApp()
	app.Name = "mp3player"
	app.Usage = "Listen to mp3 music"
	app.Description = "Use play command to play your mp3 or your playlist.\nUse playlist command to organize your playlists.\n" +
		"For more details use 'play --help' or 'playlist --help'."
	app.Commands = []cli.Command{
		{
			Name:        "play",
			Aliases:     []string{"p"},
			Usage:       "play a music or a playlist",
			ArgsUsage:   "Absolute path to your mp3",
			Description: "play \"Absolute path to your mp3\" | play pl \"Name of your playlist\"",
			Subcommands: []cli.Command{
				{
					Name:        "playlist",
					Aliases:     []string{"pl", "pll", "playl"},
					Usage:       "play your playlist",
					ArgsUsage:   "Name of your playlist from list of the playlist",
					Description: "play pl \"Name of your playlist",
					Action:      playPlaylist,
				},
			},
			Action: playSong,
		},
		{
			Name:    "playlist",
			Aliases: []string{"pl", "pll", "playl"},
			Usage:   "Organizing your playlists",
			Subcommands: []cli.Command{
				{
					Name:        "create",
					Aliases:     []string{"c", "cr"},
					Usage:       "Create a playlist",
					ArgsUsage:   "Name of your playlist",
					Description: "playlist create \"Name of the playlist\"",
					Action:      createPlaylist,
				},
				{
					Name:        "add",
					Aliases:     []string{"a", "ad"},
					Usage:       "Add a track to your playlist",
					ArgsUsage:   "Name of your playlist Absolute path to your mp3",
					Description: "playlist add \"Name of your playlist\" \"Absolut path to your mp3\"",
					Action:      addSongToPlaylist,
				},
				{
					Name:        "remove",
					Aliases:     []string{"r", "rm", "rem"},
					Usage:       "remove tracks from playlist",
					ArgsUsage:   "Name of your playlist, Name of the song",
					Description: "playlist remove \"Name of your playlist\" \"Name of the song\"",
					Action:      removeSongFromPlaylist,
				},
				{
					Name:        "delete",
					Aliases:     []string{"d", "del"},
					Usage:       "delete playlist",
					ArgsUsage:   "Name of your playlist",
					Description: "playlist delete \"Name of your playlist\"",
					Action:      deletePlaylist,
				},
				{
					Name:    "lists",
					Aliases: []string{"l", "ls"},
					Usage:   "list your playlists and songs in it",
					Action:  listPlaylists,
				},
			},
		},
	}
	app.EnableBashCompletion = true
	if _, err := os.Stat("playlists.json"); os.IsNotExist(err) {
		err := initPlaylists()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	app.Run(os.Args)
}

func initPlaylists() error {
	playlists, err := os.Create("playlists.json")
	if err != nil {
		return err
	}
	playlists.WriteString("[]")
	return nil
}

func readPlaylistsJSON(playl *[]playlist) error {
	file, err := os.Open("playlists.json")
	if err != nil {
		fmt.Println("playlists.json does not exist or damaged.", err)
		return err
	}
	fileStats, err := file.Stat()
	if err != nil {
		fmt.Println("Failing to get metadata of file.", err)
		return err
	}
	data := make([]byte, fileStats.Size())
	defer file.Close()
	file.Read(data)
	err = json.Unmarshal(data, playl)
	if err != nil {
		fmt.Println("Impossible to unmarshal json", err)
		return err
	}
	return nil
}

func updatePlaylistsJSON(playlists *[]playlist) error {
	file, err := os.Create("playlists.json")
	if err != nil {
		fmt.Println(err)
		return err
	}
	updatedPlaylists, err := json.Marshal(playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	file.Write(updatedPlaylists)
	return nil
}

func createPlaylist(c *cli.Context) error {
	if c.Args().First() == "" {
		fmt.Println("Arguments are empty.")
		return errors.New("Empty arguments.")
	}

	playlists := []playlist{}
	err := readPlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, v := range playlists {
		if v.Name == c.Args().First() {
			return fmt.Errorf("Playlist with name %v is already exist.", c.Args().First())
		}
	}
	newPlaylist := playlist{Name: c.Args().First(), Songs: []music{}}
	playlists = append(playlists, newPlaylist)
	err = updatePlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("made playlist", c.Args().First())
	return nil

}

func addSongToPlaylist(c *cli.Context) error {
	playlists := []playlist{}

	newSong, err := os.Open(c.Args().Get(1))
	if err != nil {
		fmt.Println("Could not find your mp3", err)
		return err
	}
	fileStat, err := newSong.Stat()
	if err != nil {
		fmt.Println("Error to get file metadata", err)
		return err
	}
	name := fileStat.Name()
	SongData := music{
		Name: name,
		Path: c.Args().Get(1),
	}

	err = readPlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for i := range playlists {
		if playlists[i].Name == c.Args().Get(0) {
			playlists[i].Songs = append(playlists[i].Songs, SongData)
		}
	}
	err = updatePlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Succesfully added!")
	return nil
}

func removeSongFromPlaylist(c *cli.Context) error {
	playlists := []playlist{}

	playlistName := c.Args().Get(0)
	songName := c.Args().Get(1)

	if playlistName == "" {
		return errors.New("Arguments are empty.")
	}
	if songName == "" {
		return errors.New("Missing second argument.")
	}

	err := readPlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}

	deleted := false
	for i := range playlists {
		if playlists[i].Name == playlistName {
			for j := range playlists[i].Songs {
				if playlists[i].Songs[j].Name == songName {
					playlists[i].Songs = slices.Delete(playlists[i].Songs, j, j+1)
					deleted = true
					break
				}
			}
			break
		}
	}

	err = updatePlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if deleted {
		fmt.Printf("%v successfully removed from the playlist %v", c.Args().Get(1), c.Args().Get(0))
	} else {
		fmt.Println("Something went wrong.")
	}

	return nil
}

func deletePlaylist(c *cli.Context) error {
	playlistName := c.Args().Get(0)

	if playlistName == "" {
		return errors.New("Arguments are empty.")
	}

	playlists := []playlist{}
	err := readPlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}

	for i := range playlists {
		if playlists[i].Name == playlistName {
			playlists = slices.Delete(playlists, i, i+1)
			break
		}
	}

	err = updatePlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("%v successfully deleted.", playlistName)
	return nil
}

func listPlaylists(c *cli.Context) error {
	playlists := []playlist{}
	err := readPlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, v := range playlists {
		fmt.Println("\n", v.Name)
		for _, v1 := range v.Songs {
			fmt.Println("   ", v1.Name)
		}
	}
	return nil
}

func playPlaylist(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		fmt.Println("Playlist was not found.")
		return errors.New("Arguments are empty.")
	}

	playlists := []playlist{}

	err := readPlaylistsJSON(&playlists)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var songs []string
	for i := range playlists {
		if playlists[i].Name == name {
			for j := range playlists[i].Songs {
				songs = append(songs, playlists[i].Songs[j].Path)
			}
		}
	}

	answChan := make(chan int) // Used to connect showMenu() and PlayMp3()
	defer close(answChan)
	stat := Status{false, false, false, false, beep.SampleRate(48000)}
	go showMenu(&stat, answChan)
	answChan <- 0
	i, step := 0, 0

	for i < len(songs) {
		firstPlay := true
		for stat.Loop || firstPlay {
			firstPlay = false
			step = playMp3(songs[i], answChan, &stat)
			if step == 0 {
				return nil
			}
		}
		i += step
		if i < 0 {
			break
		}
	}

	return nil
}

func playSong(c *cli.Context) error {
	path := c.Args().First()

	if path == "" {
		return errors.New("Arguments are empty.")
	}

	answChan := make(chan int)
	defer close(answChan)

	stat := Status{false, false, false, false, beep.SampleRate(48000)}
	go showMenu(&stat, answChan)
	answChan <- 0

	firstPlay := true
	for firstPlay || stat.Loop {
		firstPlay = false
		step := playMp3(path, answChan, &stat)
		if step == 0 {
			return nil
		} else if step == -1 {
			firstPlay = true //If pressed prev button restart song
		}
	}
	return nil
}

func playMp3(path string, answChan chan int, stat *Status) int {
	// Kernel of music playback, returns step for playlist playback
	// 1 - next song, -1 - prevsong, 0 - Quiting the player

	f, err := os.Open(path) // Open the music file.
	if err != nil {
		log.Fatal(err)
	}
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()
	if !stat.Inited { // Speaker should be inited only once.
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		stat.Inited = true
	}
	done := make(chan bool)                                                     // Channel to signal when the song is ended.
	resamp := beep.Resample(4, format.SampleRate, stat.oldSmapleRate, streamer) // Resample the song for proper playback.
	speaker.Play(beep.Seq(resamp, beep.Callback(func() {
		if !stat.Interupted { // If the track has ended.
			done <- true
		}
	})))
	defer speaker.Clear()

	min, sec := streamer.Len()/int(format.SampleRate)/60, streamer.Len()/int(format.SampleRate)%60 // Counting the duration of the song.
	name := strings.Split(f.Name(), "\\")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	stat.Reset()
	printedAfterPause := false //To print timer once after pause.

	for {
		select {
		case <-done:
			return 1
		case <-ticker.C: // Update timer every second.
			curMin := int(format.SampleRate.D(streamer.Position()).Round(time.Second).Seconds()) / 60
			curSec := int(format.SampleRate.D(streamer.Position()).Round(time.Second).Seconds()) % 60
			if !stat.Paused {
				fmt.Printf("\r %v ⏮️   ⏸️   ⏭️    [%v:%02v - %v:%02v]", name[len(name)-1], curMin, curSec, min, sec)
			} else if !printedAfterPause {
				fmt.Printf("\r %v ⏮️   ▶️   ⏭️    [%v:%02v - %v:%02v]", name[len(name)-1], curMin, curSec, min, sec)
				printedAfterPause = true
			}
		case answ := <-answChan: // Interaction with the menu.
			if answ == 1 { // Pause/unpause the streamer.
				if stat.Paused {
					speaker.Unlock() // Unlock speaker to continue.
				} else {
					printedAfterPause = false // Reset flag.
					speaker.Lock()            // Lock speaker to pause.
				}
				stat.Paused = !stat.Paused
				answChan <- 0
			} else if answ == 2 { // Loop/Unloop the streamer.
				stat.Loop = !stat.Loop // Updating status
				answChan <- 0

			} else if answ == 3 { // Switching to the next song in the playlist.
				stat.NextSongUpdate()
				if stat.Paused {
					speaker.Unlock()
				}
				answChan <- 0
				return 1 // Step is 1.
			} else if answ == 4 { // Switching to the previous song in the playlist.
				stat.NextSongUpdate()
				if stat.Paused {
					speaker.Unlock()
				}
				answChan <- 0
				return -1 // Step is -1.
			} else { // Quiting from the player
				if stat.Paused { // Preventing deadlock.
					speaker.Unlock()
				}
				return 0
			}
		}
	}
}

// Displays the menu and waits for user input.
func showMenu(stat *Status, answChan chan int) {
	var pause, loop string
	var answ int
	for {
		_, ok := <-answChan // Synchronizing the menu and checking if the chan is closed.
		if !ok {
			return
		}
		if stat.Paused {
			pause = "Unpause ▶️"
		} else {
			pause = "Pause ⏸️"
		}
		if stat.Loop {
			loop = "Unloop 🔁"
		} else {
			loop = "Loop 🔁"
		}
		fmt.Printf("\033[H\033[2J1.%s\n2.%s\n3.%s\n4.%s\n5.%s\n", pause, loop, "Next ⏭️", "Previous ⏮️", "Exit ⏹️") // Printing the menu.
		_, err := fmt.Scan(&answ)                                                                                   // Scanning the answer from console.
		if err != nil {
			continue
		}
		answChan <- answ // Sending the answer through the channel.
	}
}
