package view

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cyb3rplis/discord-bot-go/config"
	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"

	"github.com/bwmarrin/discordgo"
)

type Download struct {
	URL       string `json:"url"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Category  string `json:"category"`
	SoundName string `json:"sound_name"`
}

func (a *API) PromptInteractionManage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "manage":
			// sounds are shared by all servers, so only bot owners may change them
			if !a.requireOwner(s, i) {
				return
			}
			//case create, delete, move:
			if len(i.ApplicationCommandData().Options) == 0 {
				err := a.handleList(s, i)
				if err != nil {
					log.ErrorLog.Println("error handling list:", err)
				}
			} else {
				option := i.ApplicationCommandData().Options[0]
				switch option.Name {
				case "create":
					err := a.SendInteractionRespond("👉  Creating button from URL.", s, i)
					if err != nil {
						log.ErrorLog.Println("error executing buttons command:", err)
					}
					url := option.Options[0].StringValue()
					soundName := strings.TrimSpace(option.Options[1].StringValue())
					category := strings.TrimSpace(option.Options[2].StringValue())
					if err := validateName(soundName); err != nil {
						_ = a.UpdateInteractionResponse("🎶  Invalid sound name: "+err.Error(), s, i)
						return
					}
					if err := validateName(category); err != nil {
						_ = a.UpdateInteractionResponse("🎶  Invalid category: "+err.Error(), s, i)
						return
					}
					startTime := "0"
					endTime := "0"
					if len(option.Options) > 3 {
						startTime = option.Options[3].StringValue()
					}
					if len(option.Options) > 4 {
						endTime = option.Options[4].StringValue()
					}
					// Download and convert the audio
					download := Download{URL: url, Start: startTime, End: endTime, Category: category, SoundName: soundName}
					// Check if the sound directory exists, if not create it
					categoryFolder := filepath.Join(config.GetConfig().SoundsDir, download.Category)
					if _, err := os.Stat(categoryFolder); os.IsNotExist(err) {
						err = os.MkdirAll(categoryFolder, os.ModePerm)
						if err != nil {
							return
						}
					}
					err = a.DownloadAudio(download)
					if err != nil {
						log.ErrorLog.Println("error loading audio:", err)
						_ = a.UpdateInteractionResponse("🎶  Could not download audio from this URL", s, i)
						return
					}
					err = a.UpdateInteractionResponse("🎶  Audio is ready, creating button...", s, i)
					if err != nil {
						log.ErrorLog.Println("error[manage1] sending message:", err)
					}
					time.Sleep(3 * time.Second)

					err = a.ConvertMP3ToDCA(soundName, download.Category)
					if err != nil {
						log.ErrorLog.Println("error converting audio:", err)
						_ = a.UpdateInteractionResponse("🎶  Could not convert the audio", s, i)
						return
					}

					soundPath := filepath.Join(a.model.Config.SoundsDir, download.Category, download.SoundName+".dca")
					fileData, err := os.ReadFile(soundPath)
					if err != nil {
						log.WarningLog.Printf("Failed to read sound file %s: %v", soundPath, err)

						// sometimes there is a race condition where the cronjob of the sound insert is faster
						// that causes the mp3 file to be renamed to .done, which causes the dca conversion to fail
						// but the dca conversion happened already, since the mp3 was renamed.
						soundID, err := a.model.GetSoundIDByName(soundName)
						if err != nil {
							log.WarningLog.Printf("Failed to get sound ID for %s: %v", soundName, err)
							message := "🎶  Something went wrong, try again"
							err = a.UpdateInteractionResponse(message, s, i)
							if err != nil {
								log.ErrorLog.Println("error[manage2] updating interaction response:", err)
							}
							return
						}

						// sound is not in the DB, there was an actual error
						if soundID == "" {
							log.WarningLog.Printf("Sound %s not found in database", soundName)
							message := "🎶  Something went wrong, try again"
							err = a.UpdateInteractionResponse(message, s, i)
							if err != nil {
								log.ErrorLog.Println("error[manage3] updating interaction response:", err)
							}
							return
						}

						log.InfoLog.Printf("Sound %s already exists, just send response to user", soundName)
						// if we reached this point, we can assume that the sound exists already
						// Build button for the new sound
						message := "🎶  New sound added, try it out"
						component := model.BuildSingleSoundButton(soundName, category, discordgo.SuccessButton)

						err = a.UpdateInteractionResponseWithButton(message, component, s, i)
						if err != nil {
							log.ErrorLog.Printf("error[manage4] updating interaction response: %v", err)
						}

						return
					}

					fileHash, err := model.ComputeFileHash(filepath.Join(a.model.Config.SoundsDir, download.Category, download.SoundName+".mp3")) //use mp3 hash for comparing
					if err != nil {
						log.WarningLog.Printf("Failed to compute hash for file %s: %v", download.SoundName, err)
						return
					}

					var categoryID int
					existingCategories, _ := a.model.GetCategoriesM() // Get current folders/categories in DB
					existingSounds, err := a.model.GetSoundsM()
					if err != nil {
						log.ErrorLog.Println("error loading sounds:", err)
						_ = a.UpdateInteractionResponse("🎶  Database error, try again", s, i)
						return
					}
					// Check if the folder (category) exists in the database
					if dbCategoryID, exists := existingCategories[download.Category]; exists {
						categoryID = dbCategoryID // The folder already exists
					} else {
						// The folder doesn't exist in the database, so we need to add it
						if err := a.model.AddCategory(download.Category); err != nil {
							if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
								log.WarningLog.Printf("Failed to add category %s: %v", download.Category, err)
							}
							return
						}
						// Fetch the new category ID after insertion
						categoryID, err = a.model.GetCategoryByID(download.Category)
						if err != nil {
							log.ErrorLog.Println(err)
							_ = a.UpdateInteractionResponse("🎶  Database error, try again", s, i)
							return
						}
					}

					if model.FileExistsInDB(existingSounds, categoryID, download.SoundName, fileHash) {
						// File exists and hasn't changed, skip
						return
					}

					// File does not exist in the DB, add it
					if err := a.model.AddSound(categoryID, download.SoundName, fileHash, fileData); err != nil {
						//ignore this error if the sound already exists
						if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
							log.WarningLog.Printf("Failed to add sound %s to category %s: %v", download.SoundName, download.Category, err)
						}
					}
					// Build button for the new sound
					message := "🎶  New sound added, try it out"
					component := model.BuildSingleSoundButton(soundName, category, discordgo.SuccessButton)

					err = a.UpdateInteractionResponseWithButton(message, component, s, i)
					if err != nil {
						log.ErrorLog.Printf("error[manage5] updating interaction response: %v", err)
					}
					return
				case "delete":
					soundName := option.Options[0].StringValue()
					response := fmt.Sprintf("👉  Deleting button: %s", soundName)
					err := a.SendInteractionRespond(response, s, i)
					if err != nil {
						log.ErrorLog.Println("error executing buttons command:", err)
					}

					// Check if the sound exists in the database by name
					sound, err := a.model.GetSound(soundName)
					if err != nil {
						log.ErrorLog.Println("error getting sound:", err)
						return
					}
					if sound.Name == "" {
						response = fmt.Sprintf("🎶  Sound not found: %s", soundName)
						err = a.UpdateInteractionResponse(response, s, i)
						if err != nil {
							log.ErrorLog.Println("error[manage6] sending message:", err)
						}
						return
					}

					//delete file from db
					err = a.model.DeleteSound(soundName)
					if err != nil {
						log.ErrorLog.Println("error deleting sound from db:", err)
						return
					}
					response = fmt.Sprintf("🎶  Sound deleted: %s", soundName)
					err = a.UpdateInteractionResponse(response, s, i)
					if err != nil {
						log.ErrorLog.Println("error[manage7] sending message:", err)
					}
				case "move":
					soundName := option.Options[0].StringValue()
					category := strings.TrimSpace(option.Options[1].StringValue())
					if err := validateName(category); err != nil {
						_ = a.SendInteractionRespond("🎶  Invalid category: "+err.Error(), s, i)
						return
					}
					response := fmt.Sprintf("👉  Moving sound %s to category %s", soundName, category)
					err := a.SendInteractionRespond(response, s, i)
					if err != nil {
						log.ErrorLog.Println("error executing buttons command:", err)
					}

					var categoryID int
					existingCategories, _ := a.model.GetCategoriesM() // Get current folders/categories in DB

					// Check if the folder (category) exists in the database
					if dbCategoryID, exists := existingCategories[category]; exists {
						categoryID = dbCategoryID // The folder already exists
					} else {
						// The folder doesn't exist in the database, so we need to add it
						if err := a.model.AddCategory(category); err != nil {
							if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
								log.WarningLog.Printf("Failed to add category %s: %v", category, err)
							}
							return
						}
						// Fetch the new category ID after insertion
						categoryID, err = a.model.GetCategoryByID(category)
						if err != nil {
							log.ErrorLog.Println(err)
							_ = a.UpdateInteractionResponse("🎶  Database error, try again", s, i)
							return
						}
					}

					err = a.model.MoveSound(categoryID, soundName)
					if err != nil {
						log.ErrorLog.Println("error moving sound to another category:", err)
						return
					}

					response = fmt.Sprintf("🎶  Moved sound %s to category %s", soundName, category)
					err = a.UpdateInteractionResponse(response, s, i)
					if err != nil {
						log.ErrorLog.Println("error executing buttons command:", err)
					}
				default:
					err := a.SendInteractionRespond("🎶  Something went wrong...", s, i)
					if err != nil {
						log.ErrorLog.Println("fallback to default manage handler", err)
					}
				}
			}
		}
	}
}

// validName allows letters (incl. umlauts), digits, space, dash and underscore.
// It prevents path traversal ("../") and keeps button custom IDs short.
var validName = regexp.MustCompile(`^[\p{L}\p{N} _\-]{1,40}$`)

func validateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("use 1-40 letters, digits, spaces, '-' or '_'")
	}
	return nil
}

// isEmptyTimestamp treats "" and "0" as "not set".
func isEmptyTimestamp(t string) bool {
	t = strings.TrimSpace(t)
	return t == "" || t == "0"
}

// DownloadAudio downloads the audio from the provided URL.
// yt-dlp is started directly with an argument list (no shell), so user input
// like "; rm -rf /" is passed as a literal string and never executed.
func (a *API) DownloadAudio(download Download) error {
	if !strings.HasPrefix(download.URL, "https://") && !strings.HasPrefix(download.URL, "http://") {
		return fmt.Errorf("only http(s) URLs are allowed")
	}

	timeout := time.Duration(a.model.Config.AudioTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	destFile := filepath.Join(a.model.Config.SoundsDir, download.Category, download.SoundName+".mp3")
	if download.Category == "" {
		destFile = filepath.Join(a.model.Config.DataDir, download.SoundName+".mp3")
	}
	log.InfoLog.Printf("Downloading %s to %s. START: %s, END: %s", download.URL, destFile, download.Start, download.End)

	args := []string{
		"--ignore-config",        // never read yt-dlp config files from the container
		"--no-playlist",          // a playlist URL must not download dozens of videos
		"--max-filesize", "100M", // protect the disk
		"-x", "--audio-format", "mp3", "--force-overwrites", "-o", destFile,
	}
	if !isEmptyTimestamp(download.Start) || !isEmptyTimestamp(download.End) {
		start, end := "0", "inf"
		if !isEmptyTimestamp(download.Start) {
			start = strings.TrimSpace(download.Start)
		}
		if !isEmptyTimestamp(download.End) {
			end = strings.TrimSpace(download.End)
		}
		args = append(args, "--download-sections", "*"+start+"-"+end)
	}
	// "--" ends option parsing, so a URL can never be interpreted as a yt-dlp flag
	args = append(args, "--", download.URL)

	output, err := exec.CommandContext(ctx, "yt-dlp", args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		log.ErrorLog.Println("Download Audio operation timed out")
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("failed to run yt-dlp: %w, output: %s", err, string(output))
	}
	return nil
}

// ConvertMP3ToDCA converts an mp3 into the DCA (opus frame) format used for playback.
// Equivalent to `ffmpeg ... | dca > file`, but wired up in Go without a shell.
func (a *API) ConvertMP3ToDCA(fileName, categoryName string) error {
	timeout := time.Duration(a.model.Config.AudioTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sourceFile := filepath.Join(a.model.Config.SoundsDir, categoryName, fileName+".mp3")
	destFile := filepath.Join(a.model.Config.SoundsDir, categoryName, fileName+".dca")
	if categoryName == "" {
		sourceFile = filepath.Join(a.model.Config.DataDir, fileName+".mp3")
		destFile = filepath.Join(a.model.Config.DataDir, fileName+".dca")
	}

	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()

	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-loglevel", "error",
		"-i", sourceFile,
		"-filter:a", "loudnorm=I=-14:LRA=7:TP=-2, compand=attacks=0:points=-80/-80|-10/-5|0/-1",
		"-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	dca := exec.CommandContext(ctx, "dca")

	var ffErr, dcaErr bytes.Buffer
	ffmpeg.Stderr = &ffErr
	dca.Stderr = &dcaErr
	dca.Stdout = out

	// the pipe: ffmpeg stdout -> dca stdin
	pipe, err := ffmpeg.StdoutPipe()
	if err != nil {
		return err
	}
	dca.Stdin = pipe

	if err := ffmpeg.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	if err := dca.Start(); err != nil {
		_ = ffmpeg.Process.Kill()
		_ = ffmpeg.Wait()
		return fmt.Errorf("failed to start dca: %w", err)
	}
	dcaWaitErr := dca.Wait()
	ffWaitErr := ffmpeg.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		log.ErrorLog.Println("Converting Audio operation timed out")
		return ctx.Err()
	}
	if ffWaitErr != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", ffWaitErr, ffErr.String())
	}
	if dcaWaitErr != nil {
		return fmt.Errorf("dca failed: %w, output: %s", dcaWaitErr, dcaErr.String())
	}
	return nil
}
