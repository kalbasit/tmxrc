package tmux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/kalbasit/swm/code"
	"github.com/rs/zerolog/log"
)

// SwitchClient switches the TMUX to a different client
func (t *tmux) SwitchClient(killPane bool) error {
	// get all the sessions
	sessionNameProjects, err := t.getSessionNameProjects()
	if err != nil {
		return err
	}
	// select the session using fzf
	sessionName, err := t.withFilter(func(stdin io.WriteCloser) {
		for name, _ := range sessionNameProjects {
			io.WriteString(stdin, name)
			io.WriteString(stdin, "\n")
		}
	})
	if err != nil {
		return err
	}
	// get the project for the selected name
	project, ok := sessionNameProjects[sessionName]
	if !ok {
		return ErrProjectNotFoundForGivenSessionName
	}
	// make sure the project exists on disk
	if err := project.Ensure(); err != nil {
		return err
	}
	// run tmux has-session -t sessionName to check if session already exists
	if err := exec.Command(tmuxPath, "-L", t.socketName(), "has-session", "-t", sessionName).Run(); err != nil {
		// session does not exist, we should start it
		for _, args := range [][]string{
			// start the session
			{"-L", t.socketName(), "new-session", "-d", "-s", sessionName},
			// set the active profile
			{"-L", t.socketName(), "set-environment", "-t", sessionName, "ACTIVE_PROFILE", project.Story().Profile().Name()},
			// set the new GOPATH
			{"-L", t.socketName(), "set-environment", "-t", sessionName, "GOPATH", project.Story().GoPath()},
			// start a new shell on window 1
			{"-L", t.socketName(), "new-window", "-t", sessionName + ":1"},
			// start vim in the first window
			{"-L", t.socketName(), "send-keys", "-t", sessionName + ":0", "clear; vim", "Enter"},
		} {
			cmd := exec.Command(tmuxPath, args...)
			cmd.Dir = project.Path()
			// set the environment to current environment, change only ACTIVE_PROFILE and GOPATH
			cmd.Env = func() []string {
				res := []string{
					fmt.Sprintf("ACTIVE_PROFILE=%s", project.Story().Profile().Name()),
					fmt.Sprintf("GOPATH=%s", project.Story().GoPath()),
				}
				for _, v := range os.Environ() {
					if k := strings.Split(v, "=")[0]; k != "ACTIVE_PROFILE" && k != "GOPATH" && k != "TMUX" {
						res = append(res, v)
					}
				}

				return res
			}()
			// run the command now
			if err := cmd.Run(); err != nil {
				log.Fatal().Strs("args", args).Err(err).Msg("error running the tmux comand")
			}
		}
	}
	// attach the session now
	if tmuxSocketPath := os.Getenv("TMUX"); tmuxSocketPath != "" {
		// kill the pane once attached
		if killPane {
			defer func() {
				exec.Command(tmuxPath, "-L", strings.Split(path.Base(tmuxSocketPath), ",")[0], "kill-pane").Run()
			}()
		}
		return exec.Command(tmuxPath, "-L", t.socketName(), "switch-client", "-t", sessionName).Run()
	}
	cmd := exec.Command(tmuxPath, "-L", t.socketName(), "attach", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getSessionNameProjects returns a map of a project session name to the project
func (t *tmux) getSessionNameProjects() (map[string]code.Project, error) {
	sessionNameProjects := make(map[string]code.Project)

	// get the profile
	profile, err := t.options.Coder.Profile(t.options.Profile)
	if err != nil {
		return nil, err
	}
	// get the story
	story := profile.Story(t.options.Story)
	// loop over all projects and get the session name
	for _, prj := range story.Projects() {
		// generate the session name
		k := fmt.Sprintf("%s@%s=%s",
			profile.Name(),
			story.Name(),
			strings.Replace(strings.Replace(prj.ImportPath(), ".", dotChar, -1), ":", colonChar, -1))
		// assign it to the map
		sessionNameProjects[k] = prj
	}

	// get the base story
	baseStory := profile.Base()
	// loop over all base projects and get the session name
	for _, basePrj := range baseStory.Projects() {
		// if we already know about the project, skip it
		if _, ok := sessionNameProjects[basePrj.ImportPath()]; ok {
			continue
		}
		// get the project for base project from the story
		prj, err := story.Project(basePrj.ImportPath())
		if err != nil {
			return nil, err
		}
		// generate the session name
		k := fmt.Sprintf("%s@%s=%s",
			profile.Name(),
			story.Name(),
			strings.Replace(strings.Replace(prj.ImportPath(), ".", dotChar, -1), ":", colonChar, -1))
		// assign it to the map
		sessionNameProjects[k] = prj
	}

	return sessionNameProjects, nil
}
