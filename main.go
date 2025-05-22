package main

/*
This program run as endless loop until recieve SIGTERM signal. It is use to check a github repository latest commit in the specify branches.
This is useful when github webhook push method is not allowed for private service in private cloud, eg: tekton event listener is deployed in private cloud;
When this program start up, it will fetch the latest commit and store it in ${owner}-${branch}-state in current folder. Please use GitHub Rest API
Then this program will run in an interval specified by int _frequency seconds, keep checking the latest commit and top latest PR (PR_FETCH_LIMIT is the numbers to fetch).
When detect the latest commit is different from the latest-commit in file, then 
1. trigger a http post method to event_listener url with a json payload, trigger_event is "push"
2. when 1 is successful, update the ${owner}-${branch}-state file with new latest commit id, log the successful event in the stdout if it is failed for http post method, then log the error in the stdout

When detect the fetch PRs is different from the PRs in the file, then trigger a http post method to event_listener seperately
When trigger the PR change, trigger_event is "Pull Request"
Tracking PR state changes (opened/closed/updated)
Handling PR merge events separately

Error Handling: Ensure robust retry logic for GitHub API and event listener calls

The ${owner}-${branch}-state file structure is a json file:
{
    "latest_commit": "the Commit hashID",
    "PR": [
        {
            "pr_title": "",
            "pr_action": "",
            "pr_number": ""
        },
        {   "pr_title": "",
            "pr_action": "",
            "pr_number": ""
        },
        ...
    ],
    "latest_fetch_timestamp": ""
}
The json payload format
{
    "trigger_event": "",
    "ref": "refs/heads/${BRANCH}",
    "after": "${CURRENT_COMMIT_SHA}",
    "repo_url": "${REPO_URL}",
    "user_email", "${USER_EMAIL_FETCHED_FROM_GITHUB}",
    "commit_message", "${COMMIT_MESSAGE}"
    "pr_title": "",
    "pr_action": "",
    "pr_number": ""
}
 

Log requirement: Please use logrus 

Parameter are store in  config file config.json:
config.json:
{
    "OWNER": "ryanzhang",
    "REPO": "aiml-usercase",
    "BRANCH": "main",
    "EVENT_LISTENER_URL": "https://rzhang.requestcatcher.com/test",
    "FREQUENCY": 5 ,
    "PR_FETCH_LIMIT": 3,
    "LOG_LEVEL": "debug",
    "STATE_FILE_PATH": "./"

}    
GITHUB_PAT_TOKEN needs to load from env variable GITHUB_PAT_TOKEN

*/


import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Config holds application configuration
type Config struct {
	Owner            string `json:"OWNER"`
	Repo             string `json:"REPO"`
	Branch           string `json:"BRANCH"`
	EventListenerURL string `json:"EVENT_LISTENER_URL"`
	Frequency        int    `json:"FREQUENCY"`
	PRFetchLimit     int    `json:"PR_FETCH_LIMIT"`
	LogLevel         string `json:"LOG_LEVEL"`
	StateFilePath    string `json:"STATE_FILE_PATH"`
}

// StateFile represents the persistent state
type CommitStateFile struct {
	LatestCommit        string    `json:"latest_commit"`
	LatestFetchTimestamp string   `json:"latest_fetch_timestamp"`
}

// StateFile represents the persistent state
type PRStateFile struct {
	PRs                 []PRInfo  `json:"PR"`
	LatestFetchTimestamp string   `json:"latest_fetch_timestamp"`
}

// PRInfo holds PR information
type PRInfo struct {
	Title  string `json:"pr_title"`
	Action string `json:"pr_action"`
	Number int    `json:"pr_number"`
	State  string `json:"pr_state"` // "open", "closed", "merged"
    PRBranch string `json:"pr_branch"`
    PRCommitId string `json:"pr_commit_id"`
}

// EventPayload is sent to the event listener
type EventPayload struct {
	TriggerEvent   string `json:"trigger_event"`
	Ref            string `json:"ref"`
	After          string `json:"after"`
	RepoURL        string `json:"repo_url"`
	UserEmail      string `json:"user_email"`
	CommitMessage  string `json:"commit_message"`
	PRTitle        string `json:"pr_title,omitempty"`
	PRAction       string `json:"pr_action,omitempty"`
	PRNumber       int    `json:"pr_number,omitempty"`
	PRState        string `json:"pr_state,omitempty"`
    CloneURL string `json:"clone_url,omitempty"`
    PRBranch string `json:"pr_branch,omitempty"`
}

var (
	log        = logrus.New()
	config     Config
	commitStateFile  string
	prStateFile  string
	httpClient = &http.Client{Timeout: 10 * time.Second}
	githubPAT  string
)

func main() {
	// Load GitHub PAT from environment
	githubPAT = os.Getenv("GITHUB_PAT_TOKEN")
	if githubPAT == "" {
		log.Fatal("GITHUB_PAT_TOKEN environment variable is required")
	}

	// Initialize logging
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	initLogging()

	// Initialize state file path
	commitStateFile = filepath.Join(config.StateFilePath, fmt.Sprintf("%s-%s-commit-state.json", config.Owner, config.Branch))
	prStateFile = filepath.Join(config.StateFilePath, fmt.Sprintf("%s-%s-pr-state.json", config.Owner, config.Branch))

	// Set up GitHub client
	client := createGitHubClient(githubPAT)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandling(cancel)

	// Main monitoring loop
	ticker := time.NewTicker(time.Duration(config.Frequency) * time.Second)
	defer ticker.Stop()

	log.Info("Starting GitHub commit & PR monitor")

	// Initial state check
	if err := checkForUpdates(ctx, client); err != nil {
		log.Errorf("Initial check failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down...")
			return
		case <-ticker.C:
			if err := checkForUpdates(ctx, client); err != nil {
				log.Errorf("Update check failed: %v", err)
			}
		}
	}
}

func initLogging() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)

	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
}

func loadConfig(filename string) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Set defaults
	if config.PRFetchLimit == 0 {
		config.PRFetchLimit = 3
	}
	if config.Frequency == 0 {
		config.Frequency = 5
	}
	if config.StateFilePath == "" {
		config.StateFilePath = "."
	}

	return nil
}

func createGitHubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}

func setupSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal: %v", sig)
		cancel()
	}()
}

func checkForUpdates(ctx context.Context, client *github.Client) error {
	// Load current commit state
	currentCommitState, err := loadCommitState()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Check for commit changes
	newCommit, commitMessage, err := getLatestCommit(ctx, client)
	if err != nil {
		return fmt.Errorf("getting latest commit: %w", err)
	}

	if newCommit != currentCommitState.LatestCommit {
		if err := handleCommitChange(ctx, client, newCommit, commitMessage); err != nil {
			return fmt.Errorf("handling commit change: %w", err)
		}
	}

    // Load current commit state
	currentPRState, err := loadPRState()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	// Check for PR changes
	newPRs, err := getLatestPRs(ctx, client)
	if err != nil {
		return fmt.Errorf("getting latest PRs: %w", err)
	}

	if prChanges := comparePRs(currentPRState.PRs, newPRs); len(prChanges) > 0 {
		if err := handlePRChanges(ctx, client, prChanges); err != nil {
			return fmt.Errorf("handling PR changes: %w", err)
		}
	}

	// // Update fetch timestamp
	// currentCommitState.LatestFetchTimestamp = time.Now().Format(time.RFC3339)
	// if err := saveCommitState(currentCommitState); err != nil {
	// 	return fmt.Errorf("saving state: %w", err)
	// }

	return nil
}

func loadCommitState() (*CommitStateFile, error) {
	var state CommitStateFile

	if _, err := os.Stat(commitStateFile); os.IsNotExist(err) {
		return &state, nil
	}

	data, err := ioutil.ReadFile(commitStateFile)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return &state, nil
}

func saveCommitState(state *CommitStateFile) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := ioutil.WriteFile(commitStateFile, data, 0644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}


func loadPRState() (*PRStateFile, error) {
	var state PRStateFile

	if _, err := os.Stat(prStateFile); os.IsNotExist(err) {
		return &state, nil
	}

	data, err := ioutil.ReadFile(prStateFile)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return &state, nil
}

func savePRState(state *PRStateFile) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := ioutil.WriteFile(prStateFile, data, 0644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}

func getLatestCommit(ctx context.Context, client *github.Client) (string, string, error) {
	ref, _, err := client.Git.GetRef(ctx, config.Owner, config.Repo, "refs/heads/"+config.Branch)
	if err != nil {
		return "", "", fmt.Errorf("getting branch reference: %w", err)
	}

	commit, _, err := client.Git.GetCommit(ctx, config.Owner, config.Repo, ref.GetObject().GetSHA())
	if err != nil {
		return "", "", fmt.Errorf("getting commit details: %w", err)
	}

	return commit.GetSHA(), commit.GetMessage(), nil
}

func getLatestPRs(ctx context.Context, client *github.Client) ([]PRInfo, error) {
	prs, _, err := client.PullRequests.List(ctx, config.Owner, config.Repo, &github.PullRequestListOptions{
		State: "all",
		Sort:  "updated",
		ListOptions: github.ListOptions{
			PerPage: config.PRFetchLimit,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}

	var result []PRInfo
	for _, pr := range prs {
		action := "updated"
		if pr.GetCreatedAt().Equal(pr.GetUpdatedAt()) {
			action = "opened"
		}

		state := pr.GetState()
		if pr.GetMerged() {
			state = "merged"
		}

		result = append(result, PRInfo{
			Title:  pr.GetTitle(),
			Action: action,
			Number: pr.GetNumber(),
			State:  state,
            PRBranch:   pr.GetHead().GetRef(),          // Get the branch name
            PRCommitId: pr.GetHead().GetSHA(),          // Get the latest commit SHA
		})
	}

	return result, nil
}

func comparePRs(oldPRs []PRInfo, newPRs []PRInfo) []PRInfo {
	var changes []PRInfo

	// Check for new or updated PRs
	for _, newPR := range newPRs {
		found := false
		for _, oldPR := range oldPRs {
			if newPR.Number == oldPR.Number {
				if newPR.Action != oldPR.Action || newPR.State != oldPR.State || 
                    newPR.PRCommitId != oldPR.PRCommitId ||
                    newPR.PRBranch != oldPR.PRBranch{
					changes = append(changes, newPR)
				}
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, newPR)
		}
	}

	// Check for closed PRs not in new list
	for _, oldPR := range oldPRs {
		found := false
		for _, newPR := range newPRs {
			if oldPR.Number == newPR.Number {
				found = true
				break
			}
		}
		if !found && oldPR.State != "closed" {
			oldPR.Action = "closed"
			oldPR.State = "closed"
			changes = append(changes, oldPR)
		}
	}

	return changes
}

func handleCommitChange(ctx context.Context, client *github.Client, newCommit, commitMessage string) error {
	// Get commit author email
	commit, _, err := client.Git.GetCommit(ctx, config.Owner, config.Repo, newCommit)
	if err != nil {
		return fmt.Errorf("getting commit author: %w", err)
	}

	userEmail := commit.GetAuthor().GetEmail()
	if userEmail == "" {
		userEmail = "unknown@github.com"
	}

	payload := EventPayload{
		TriggerEvent:  "push",
		Ref:           fmt.Sprintf("refs/heads/%s", config.Branch),
		After:         newCommit,
		RepoURL:       fmt.Sprintf("https://github.com/%s/%s", config.Owner, config.Repo),
		UserEmail:     userEmail,
		CommitMessage: commitMessage,
	}

	if err := triggerEventListener(payload); err != nil {
		return fmt.Errorf("triggering event listener: %w", err)
	}

	// Update state
	state, err := loadCommitState()
	if err != nil {
		return fmt.Errorf("loading state for update: %w", err)
	}

	state.LatestCommit = newCommit
    state.LatestFetchTimestamp = time.Now().UTC().Format(time.RFC3339)
	if err := saveCommitState(state); err != nil {
		return fmt.Errorf("saving updated state: %w", err)
	}

	log.Infof("Successfully processed commit change: %s", newCommit)
	return nil
}

func handlePRChanges(ctx context.Context, client *github.Client, prChanges []PRInfo) error {
	for _, pr := range prChanges {
        // Get the full PR info including merge commit SHA
        fullPR, _, err := client.PullRequests.Get(ctx, config.Owner, config.Repo, pr.Number)
        if err != nil {
            log.Errorf("Failed to get PR details for #%d: %v", pr.Number, err)
            continue
        }

        // Get the head commit SHA (latest commit in the PR)
        headSHA := fullPR.GetHead().GetSHA()
        if headSHA == "" {
            log.Errorf("No head SHA found for PR #%d", pr.Number)
            continue
        }

        // Get the clone URL for the PR branch
        cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", fullPR.GetHead().GetUser().GetLogin(), fullPR.GetHead().GetRepo().GetName())
        prBranch := fullPR.GetHead().GetRef()


        // Get user email
        userEmail := fullPR.GetUser().GetEmail()
        if userEmail == "" {
            userEmail = "unknown@github.com"
        }

		// payload := EventPayload{
		// 	TriggerEvent:  "Pull Request",
		// 	RepoURL:       fmt.Sprintf("https://github.com/%s/%s", config.Owner, config.Repo),
		// 	UserEmail:     userEmail,
		// 	PRTitle:       pr.Title,
		// 	PRAction:      pr.Action,
		// 	PRNumber:      pr.Number,
		// 	PRState:       pr.State,
		// 	CommitMessage: fmt.Sprintf("PR #%d %s: %s", pr.Number, pr.Action, pr.Title),
		// }
        // Build the payload with PR commit information
        payload := EventPayload{
            TriggerEvent:  "Pull Request",
            RepoURL:       fmt.Sprintf("https://github.com/%s/%s", config.Owner, config.Repo),
            UserEmail:     userEmail,
            PRTitle:       pr.Title,
            PRAction:      pr.Action,
            PRNumber:      pr.Number,
            PRState:       pr.State,
            Ref:           fmt.Sprintf("refs/pull/%d/head", pr.Number), // Special ref for PR head
            After:         headSHA, // Latest commit SHA in the PR
            CommitMessage: fmt.Sprintf("PR #%d %s: %s", pr.Number, pr.Action, pr.Title),
            CloneURL:      cloneURL, // URL to clone the PR repo
            PRBranch:      prBranch, // Branch name in the source repo
        }

		if err := triggerEventListener(payload); err != nil {
			log.Errorf("Failed to trigger event for PR #%d: %v", pr.Number, err)
			continue
		}

		log.Infof("Successfully processed PR change: #%d %s", pr.Number, pr.Action)
	}

	// Update state with all PRs (not just changes)
	newPRs, err := getLatestPRs(ctx, client)
	if err != nil {
		return fmt.Errorf("getting latest PRs for state update: %w", err)
	}

	state, err := loadPRState()
	if err != nil {
		return fmt.Errorf("loading state for PR update: %w", err)
	}

	state.PRs = newPRs
    state.LatestFetchTimestamp = time.Now().UTC().Format(time.RFC3339)
	if err := savePRState(state); err != nil {
		return fmt.Errorf("saving updated PR state: %w", err)
	}

	return nil
}

func triggerEventListener(payload EventPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}
    // Log the payload being sent (debug level)
    log.WithFields(logrus.Fields{
        "url":     config.EventListenerURL,
        "payload": string(payloadBytes),
    }).Debug("Sending HTTP POST request to event listener")

	// With retry logic
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := httpClient.Post(config.EventListenerURL, "application/json", bytes.NewReader(payloadBytes))
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := ioutil.ReadAll(resp.Body)
			lastErr = fmt.Errorf("event listener returned error: %s - %s", resp.Status, string(body))
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return nil
	}

	return lastErr
}
