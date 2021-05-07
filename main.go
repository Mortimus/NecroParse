package main

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	everquest "github.com/Mortimus/goEverquest"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	htgotts "github.com/hegedustibor/htgo-tts"
)

var Debug, Warn, Err, Info *log.Logger
var SpellDB everquest.SpellDB
var activeMob binding.String

func init() {
	// Initialize log handlers
	LogFile, err := os.OpenFile(configuration.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	Warn = log.New(LogFile, "[WARN] ", log.Lshortfile|log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)
	Err = log.New(LogFile, "[ERR] ", log.Lshortfile|log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)
	Info = log.New(LogFile, "[INFO] ", log.Lshortfile|log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)
	Debug = log.New(LogFile, "[DEBUG] ", log.Lshortfile|log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)
	if configuration.Log.Level < 0 {
		Warn.SetOutput(ioutil.Discard)
	}
	if configuration.Log.Level < 1 {
		Err.SetOutput(ioutil.Discard)
	}
	if configuration.Log.Level < 2 {
		Info.SetOutput(ioutil.Discard)
	}
	if configuration.Log.Level < 3 {
		Debug.SetOutput(ioutil.Discard)
	}
	SpellDB.LoadFromFile(configuration.Everquest.SpellDB, Err)
}

func main() {
	Debug.Printf("Using Config\n%#+v", configuration)
	// kiss := findSpellIDByName("Saryrn's Kiss")
	// kissS := SpellDB.GetSpellByID(kiss)
	// fmt.Printf("Kiss\n%#+v\n", kissS)
	// return
	ChatLogs := make(chan everquest.EqLog)
	go everquest.BufferedLogRead(configuration.Everquest.LogPath, configuration.Main.ReadEntireLog, configuration.Main.LogPollRate, ChatLogs)
	myApp := app.New()
	myWindow := myApp.NewWindow("NecroParse")

	activeMob = binding.NewString()
	activeMob.Set("None")
	mobLabel := widget.NewLabelWithData(activeMob)

	content := container.NewVBox(
		mobLabel, // We need data binding for this
	)
	go parseLogs(content, ChatLogs)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func parseLogs(ui *fyne.Container, logs chan everquest.EqLog) {
	for l := range logs { // handle all logs in the channel
		if l.Channel == "system" { // Spells are "system" level logs, so we can drop the rest
			processLog(ui, l)
		}
	}
}

// Gangrenous Touch of Zum'uul
// Gangrenous Touch of Zum`uul
// TODO: update everquest package to change ` to '

func processLog(ui *fyne.Container, l everquest.EqLog) {
	if configuration.Main.ReadEntireLog { // slow things down so we can watch parser working
		time.Sleep(1 * time.Millisecond)
	}
	// TODO: update mob name based on cast on other text if trigger time+spell cast time(include haste) -- we'll have to account for that cleric haste spell too
	// Debug.Printf("Processing: %s\n", l.Msg)
	r, _ := regexp.Compile(configuration.Everquest.TakenDamageRegex)
	result := r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		activeMob.Set(result[1])
		Debug.Printf("Handling damage taken: %s", l.Msg)
		spellID := findSpellIDByName(result[3])
		if spellID != -1 { // we found the spell
			spell := SpellDB.GetSpellByID(spellID)
			if spell.Spelltype != "Detrimental" || spell.Duration <= 0 {
				Debug.Printf("Non long term Detrimental spell, ignoring: %d:%s", spell.Id, spell.Name)
				return
			}
			if _, ok := active[spell.Name]; ok {
				index := findSpellInstance(result[1], spell.Name)
				if index >= 0 {
					if (*active[spell.Name])[index].target != "Unknown" {
						Debug.Printf("Handling first tick of spell %s on %s", spell.Name, result[1])
						// Check if an unknown exists for this spell, if so reset ticks and remove the unknown
						uIndex := findSpellInstance("Unknown", spell.Name)
						if uIndex >= 0 {
							Debug.Printf("Removing old instances of %s", spell.Name)
							(*active[spell.Name])[index].ticks = 0
							removeInstance(uIndex, spell.Name)
							index = findSpellInstance(result[1], spell.Name) // need to re-find instance
							if index < 0 {
								Err.Printf("Cannot find index after removing old instances")
								return
							}
						}
					}
					(*active[spell.Name])[index].target = result[1]
					(*active[spell.Name])[index].tick()
				}
			}
		} else {
			Err.Printf("Cannot find spell %s\n", result[3])
		}
		return
	}
	// You begin casting (.+).
	r, _ = regexp.Compile(configuration.Everquest.BeginCastingRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		Debug.Printf("Handling begin casting: %s", l.Msg)
		spellID := findSpellIDByName(result[1])
		if spellID != -1 { // we found the spell
			spell := SpellDB.GetSpellByID(spellID)
			if spell.Spelltype != "Detrimental" || spell.Duration <= 0 {
				Debug.Printf("Non long term Detrimental spell, ignoring: %d:%s", spell.Id, spell.Name)
				return
			}
			a := &ActiveSpell{}
			a.set(ui, spell)
			if active[spell.Name] == nil {
				act := &[]ActiveSpell{}
				active[spell.Name] = act
			}
			*active[spell.Name] = append(*active[spell.Name], *a)
		} else {
			Err.Printf("Cannot find spell %s\n", result[1])
		}
		return
	}
	// Your (.+) spell is interrupted.
	r, _ = regexp.Compile(configuration.Everquest.InterruptedRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		Debug.Printf("Handling spell interrupted: %s", l.Msg)
		if _, ok := active[result[1]]; ok {
			index := findSpellInstance("Unknown", result[1])
			if index >= 0 {
				(*active[result[1]])[index].cleanup()
				removeInstance(index, result[1])
			}
		}
		return
	}
	// Your (.+) spell fizzles!
	r, _ = regexp.Compile(configuration.Everquest.FizzledRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		Debug.Printf("Handling fizzle: %s", l.Msg)
		if _, ok := active[result[1]]; ok {
			index := findSpellInstance("Unknown", result[1])
			if index >= 0 {
				(*active[result[1]])[index].cleanup()
				removeInstance(index, result[1])
			}
		}
		return
	}
	// Your (.+) spell has worn off of (.*).
	r, _ = regexp.Compile(configuration.Everquest.WornOffRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		activeMob.Set(result[2])
		Debug.Printf("Handling worn off: %s", l.Msg)
		if _, ok := active[result[1]]; ok {
			index := findSpellInstance(result[2], result[1])
			if index >= 0 {
				(*active[result[1]])[index].cleanup()
				removeInstance(index, result[1])
			}
		}
		// TODO: Alert the spell has worn off
		return
	}
	// Your (.+) spell on (.+) has been overwritten.
	r, _ = regexp.Compile(configuration.Everquest.OverwrittenRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		activeMob.Set(result[2])
		Debug.Printf("Handling overwritten: %s", l.Msg)
		if _, ok := active[result[1]]; ok {
			index := findSpellInstance(result[2], result[1])
			if index >= 0 {
				(*active[result[1]])[index].cleanup()
				removeInstance(index, result[1])
			}
		}
		// TODO: Alert the spell has been overwritten
		return
	}
	// (.+) has been slain by (.+)!
	r, _ = regexp.Compile(configuration.Everquest.KilledRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		// TODO: Account for multiple mobs
		found := cleanupMob(result[1])
		if found > 0 { // Only update current target if we had a dot on the now dead mob
			Debug.Printf("Handling npc death: %s", l.Msg)
			activeMob.Set("None")
		}
		return
	}
	// (You) have been slain by (.+)!
	r, _ = regexp.Compile(configuration.Everquest.DiedRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		Debug.Printf("Handling self death: %s", l.Msg)
		cleanupAll()
		activeMob.Set("None")
		return
	}
	// (.+) resisted your (.+)!
	r, _ = regexp.Compile(configuration.Everquest.ResistedRegex)
	result = r.FindStringSubmatch(l.Msg)
	if len(result) > 0 {
		activeMob.Set(result[1])
		Debug.Printf("Handling resist: %s", l.Msg)
		if _, ok := active[result[2]]; ok {
			index := findSpellInstance("Unknown", result[2])
			if index >= 0 {
				if active[result[1]] != nil {
					(*active[result[1]])[index].cleanup()
					removeInstance(index, result[1])
				}
			}
		}
		// TODO: Alert the spell has been resisted
		return
	}
}

func findSpellIDByName(name string) int {
	// we need to check against overrides
	for _, o := range configuration.Overrides {
		if o.Name == name {
			return o.SpellID
		}
	}
	return SpellDB.FindIDByName(name)
}

func playAudio(say string) {
	// Check if the audio exists, otherwise create it with tts
	if _, err := os.Stat("audio" + "/" + say + ".mp3"); os.IsNotExist(err) {
		Debug.Printf("Creating mp3 for %s", say)
		speech := htgotts.Speech{Folder: "audio", Language: "en"}
		err := speech.Speak(say)
		if err != nil {
			Err.Println(err)
		}
	}
	hash := md5.Sum([]byte(say))
	name := hex.EncodeToString(hash[:]) // tts uses hashes now, so we have to find the hash name
	f, err := os.Open("audio" + "/" + name + ".mp3")
	if err != nil {
		Err.Fatalln(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		Err.Fatalln(err)
	}
	defer streamer.Close()
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		Err.Fatalln(err)
	}
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}
