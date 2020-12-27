package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	htgotts "github.com/hegedustibor/htgo-tts"
)

// Spells is all known necro spell
var Spells []Spell
var activeSpells = make(map[string]int)
var tarMob string

func main() {
	// Open Configuration and set log output
	configFile, err := os.OpenFile(configuration.LogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer configFile.Close()
	log.SetOutput(configFile)
	l := LogInit("main-main.go")
	defer l.End()
	seedSpells()
	bufferedRead(configuration.EQLogPath, configuration.ReadEntireLog)
}

func bufferedRead(path string, fromStart bool) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("error opening buffered file: %v", err)
	}
	if !fromStart {
		file.Seek(0, 2) // move to end of file
	}
	bufferedReader := bufio.NewReader(file)
	r, _ := regexp.Compile(configuration.EQBaseLogLine)
	for {
		str, err := bufferedReader.ReadString('\n')
		if err == io.EOF {
			time.Sleep(time.Duration(configuration.LogPollRate) * time.Second) // 1 eq tick = 6 seconds
			continue
		}
		if err != nil {
			log.Fatalf("error opening buffered file: %v", err)
		}

		results := r.FindAllStringSubmatch(str, -1) // this really needs converted to single search
		if results == nil {
			time.Sleep(3 * time.Second)
		} else {
			t := eqTimeConv(results[0][1])
			msg := strings.TrimSuffix(results[0][2], "\r")
			l := &EqLog{
				t:       t,
				msg:     msg,
				channel: getChannel(msg),
			}
			parseLogLine(*l)
		}
	}
}

func eqTimeConv(t string) time.Time {
	// Get local time zone
	localT := time.Now()
	zone, _ := localT.Zone()
	// fmt.Println(zone, offset)

	// Parse Time
	cTime, err := time.Parse("Mon Jan 02 15:04:05 2006 MST", t+" "+zone)
	if err != nil {
		fmt.Printf("Error parsing time, defaulting to now: %s\n", err.Error())
		cTime = time.Now()
	}
	return cTime
}

// EqLog represents a single line of eq logging
type EqLog struct {
	t       time.Time
	msg     string
	channel string
}

func getChannel(msg string) string {
	m := strings.Split(msg, " ")
	if len(m) > 1 && m[1] == "tells" {
		// return m[3]
		return strings.TrimRight(m[3], ",")
	}
	return "system"
}

func parseLogLine(l EqLog) {

	if l.channel == "system" {
		r, _ := regexp.Compile(`(.+) has taken (\d+) damage from your (.+).`)
		result := r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			if checkKnownSpell(result[3]) {
				if val, ok := activeSpells[result[3]]; ok {
					activeSpells[result[3]] = val + 1
				}
				if (getMaxTicks(result[3]) - activeSpells[result[3]]) <= configuration.TickAlert {
					fmt.Printf("WARNING: %s has %d tick left\n", result[3], getMaxTicks(result[3])-activeSpells[result[3]])
					playAudio(result[3])
				}
				tarMob = result[1]
			}
			return
		}
		// You begin casting (.+).
		r, _ = regexp.Compile(`You begin casting (.+).`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			if checkKnownSpell(result[1]) {
				// fmt.Printf("%s: %#v\n", l.t.String(), result)
				activeSpells[result[1]] = 0
			}
			return
		}
		// Your (.+) spell is interrupted.
		r, _ = regexp.Compile(`Your (.+) spell is interrupted.`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			if checkKnownSpell(result[1]) {
				if _, ok := activeSpells[result[1]]; ok {
					delete(activeSpells, result[1])
				}
			}
			return
		}
		// Your (.+) spell fizzles!
		r, _ = regexp.Compile(`Your (.+) spell fizzles!`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			if checkKnownSpell(result[1]) {
				if _, ok := activeSpells[result[1]]; ok {
					delete(activeSpells, result[1])
				}
			}
			return
		}
		// Your (.+) spell has worn off of (.*).
		r, _ = regexp.Compile(`Your (.+) spell has worn off of (.*).`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			if checkKnownSpell(result[1]) {
				if _, ok := activeSpells[result[1]]; ok {
					delete(activeSpells, result[1])
				}
				playAudio(result[1])
				tarMob = result[2]
			}
			return
		}
		// Your (.+) spell on (.+) has been overwritten.
		r, _ = regexp.Compile(`Your (.+) spell on (.+) has been overwritten.`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			if checkKnownSpell(result[1]) {
				if _, ok := activeSpells[result[1]]; ok {
					delete(activeSpells, result[1])
				}
				fmt.Printf("WARNING: %s has been overwritten\n", result[1])
				playAudio(result[1])
				tarMob = result[2]
			}
			return
		}
		// (.+) has been slain by (.+)!
		r, _ = regexp.Compile(`(.+) has been slain by (.+)!`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			// TODO: Clear map if mob is the one we were dotting
			if strings.Compare(tarMob, result[1]) == 0 {
				fmt.Printf("Done[%s]: %#v\n", result[1], activeSpells)
				activeSpells = make(map[string]int)
			}
			return
		}
		// (You) have been slain by (.+)!
		r, _ = regexp.Compile(`(You) have been slain by (.+)!`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			activeSpells = make(map[string]int)
			return
		}
		// (.+) resisted your (.+)!
		r, _ = regexp.Compile(`(.+) resisted your (.+)!`)
		result = r.FindStringSubmatch(l.msg)
		if len(result) > 0 {
			// fmt.Printf("%s: %#v\n", l.t.String(), result)
			if checkKnownSpell(result[2]) {
				if _, ok := activeSpells[result[2]]; ok {
					delete(activeSpells, result[2])
				}
				tarMob = result[1]
			}
			return
		}
	}
}

// Spell defines a spell and its statistics
type Spell struct {
	name     string
	castTime float32
	ticks    int
	reuse    float32
	fizzle   float32
}

func seedSpells() {
	s := Spell{
		name:     "Ancient: Lifebane",
		castTime: 5.0,
		ticks:    0,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Trucidation",
		castTime: 3.2,
		ticks:    0,
		reuse:    900,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Gangrenous Touch of Zum`uul",
		castTime: 3.2,
		ticks:    0,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Devouring Darkness",
		castTime: 3,
		ticks:    13,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Pyrocruor",
		castTime: 3,
		ticks:    8,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Exile Undead",
		castTime: 4.5,
		ticks:    0,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Vexing Mordinia",
		castTime: 5.5,
		ticks:    9,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Cessation of Cor",
		castTime: 3,
		ticks:    9,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Conglaciation of Bone",
		castTime: 6,
		ticks:    0,
		reuse:    12,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Chill Bones",
		castTime: 6,
		ticks:    0,
		reuse:    12,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Touch of Night",
		castTime: 3.2,
		ticks:    0,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Plague",
		castTime: 3,
		ticks:    13,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Defoliation",
		castTime: 5,
		ticks:    0,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Splurt",
		castTime: 3,
		ticks:    16,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Envenomed Bolt",
		castTime: 3,
		ticks:    6,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Ignite Blood",
		castTime: 3,
		ticks:    7,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Bond of Death",
		castTime: 7,
		ticks:    9,
		reuse:    10,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Vampiric Curse",
		castTime: 4,
		ticks:    9,
		reuse:    10,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Boil Blood",
		castTime: 3,
		ticks:    7,
		reuse:    1.5,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Heat Blood",
		castTime: 3,
		ticks:    6,
		reuse:    4,
		fizzle:   1.5,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Morternum",
		castTime: 10,
		ticks:    9,
		reuse:    0,
		fizzle:   0,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Soul Well",
		castTime: 13.5,
		ticks:    9,
		reuse:    0,
		fizzle:   0,
	}
	Spells = append(Spells, s)
	s = Spell{
		name:     "Torment of Shadows",
		castTime: 9,
		ticks:    16,
		reuse:    0,
		fizzle:   0,
	}
	Spells = append(Spells, s)
}

func checkKnownSpell(s string) bool {
	for _, spell := range Spells {
		// fmt.Printf("KnownSpell: %s UnknownSpell: %s\n", spell.name, s)
		if strings.Compare(spell.name, s) == 0 {
			return true
		}
	}
	// fmt.Printf("Unknown Spell: %s\n", s)
	return false
}

func getMaxTicks(s string) int {
	for _, spell := range Spells {
		if strings.Compare(spell.name, s) == 0 {
			return spell.ticks
		}
	}
	return 0
}

func playAudio(say string) {
	// Check if the audio exists, otherwise create it with tts
	if _, err := os.Stat("audio" + "/" + say + ".mp3"); os.IsNotExist(err) {
		speech := htgotts.Speech{Folder: "audio", Language: "en"}
		speech.Speak(say)
	}
	f, err := os.Open("audio" + "/" + say + ".mp3")
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}
