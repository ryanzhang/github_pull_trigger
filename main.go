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
    ]
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
 

Log requirement:
    Please use logrus 

Parameter are store in  config file config.json:
config.json:
{
    "OWNER": "ryanzhang",
    "REPO": "aiml-usercase",
    "BRANCH": "main",
    "EVENT_LISTENER_URL": "https://rzhang.requestcatcher.com/test",
    "FREQUENCY": 5 ,
    "PR_FETCH_LIMIT": 3 

}    

*/


import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"

    "github.com/google/go-github/v58/github"
    "golang.org/x/oauth2"
)

const (
    commitFile = "latest-commit"
)

type Payload struct {
    Ref           string `json:"ref"`
    After         string `json:"after"`
    RepoURL       string `json:"repo_url"`
    UserEmail     string `json:"user_email"`
    CommitMessage string `json:"commit_message"`
}

type Config struct {
    Owner            string
    Repo             string
    Branch           string
    EventListenerURL string
    Frequency        int
    UserEmail        string
    GitHubToken      string
}

func createGitHubClient(token string) *github.Client {
    if token == "" {
        return github.NewClient(nil) // Unauthenticated client (60 req/hour limit)
    }

    // Create authenticated client with PAT
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )
    tc := oauth2.NewClient(ctx, ts)

    return github.NewClient(tc)
}

func loadConfig() (*Config, error) {
    config := &Config{
        Owner:            os.Getenv("OWNER"),
        Repo:             os.Getenv("REPO"),
        Branch:           os.Getenv("BRANCH"),
        EventListenerURL: os.Getenv("EVENT_LISTENER_URL"),
        // UserEmail:        os.Getenv("USER_EMAIL"),
        GitHubToken:      os.Getenv("GITHUB_PAT_TOKEN"),
    }

    if config.Owner == "" || config.Repo == "" || config.Branch == "" {
        return nil, fmt.Errorf("OWNER, REPO, and BRANCH environment variables are required")
    }

    if config.EventListenerURL == "" {
        return nil, fmt.Errorf("EVENT_LISTENER_URL environment variable is required")
    }

    freqStr := os.Getenv("FREQUENCY")
    if freqStr == "" {
        config.Frequency = 60 // default to 60 seconds
    } else {
        freq, err := strconv.Atoi(freqStr)
        if err != nil {
            return nil, fmt.Errorf("FREQUENCY must be a number: %v", err)
        }
        config.Frequency = freq
    }

    // if config.UserEmail == "" {
    //     config.UserEmail = "no_useremail@example.com"
    // }

    return config, nil
}

func getLatestCommitID(client *github.Client, owner, repo, branch string) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    ref, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
    if err != nil {
        return "", fmt.Errorf("failed to get branch reference: %v", err)
    }
    

    return ref.GetObject().GetSHA(), nil
}

func readStoredCommit() (string, error) {
    if _, err := os.Stat(commitFile); os.IsNotExist(err) {
        return "", nil
    }

    data, err := os.ReadFile(commitFile)
    if err != nil {
        return "", fmt.Errorf("failed to read commit file: %v", err)
    }

    return string(data), nil
}

func storeCommit(commitID string) error {
    return os.WriteFile(commitFile, []byte(commitID), 0644)
}

func triggerEventListener(url string, payload Payload) error {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %v", err)
    }

    resp, err := http.Post(url, "application/json", bytes.NewReader(payloadBytes))
    if err != nil {
        return fmt.Errorf("HTTP request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("event listener returned error: %s - %s", resp.Status, string(body))
    }

    return nil
}

func main() {
    config, err := loadConfig()
    if err != nil {
        fmt.Printf("Error loading configuration: %v\n", err)
        os.Exit(1)
    }

    // Create GitHub client
    client := createGitHubClient(config.GitHubToken)

    // Handle Ctrl+C
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Initial fetch
    initialCommit, err := getLatestCommitID(client, config.Owner, config.Repo, config.Branch)
    if err != nil {
        fmt.Printf("Error getting initial commit: %v\n", err)
        os.Exit(1)
    }

    err = storeCommit(initialCommit)
    if err != nil {
        fmt.Printf("Error storing initial commit: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Initial commit stored: %s\n", initialCommit)

    // Main loop
    fmt.Printf("Fetch %s/%s at %s branch every %d seconds\n", config.Owner, config.Repo, config.Branch, config.Frequency)
    ticker := time.NewTicker(time.Duration(config.Frequency) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-sigChan:
            fmt.Println("\nReceived termination signal. Exiting...")
            return

        case <-ticker.C:
            currentCommit, err := readStoredCommit()
            if err != nil {
                fmt.Printf("Error reading stored commit: %v\n", err)
                continue
            }

            latestCommit, err := getLatestCommitID(client, config.Owner, config.Repo, config.Branch)
            if err != nil {
                fmt.Printf("Error getting latest commit: %v\n", err)
                continue
            }

            if latestCommit != currentCommit {
                fmt.Printf("New commit detected: %s (was: %s)\n", latestCommit, currentCommit)

                payload := Payload{
                    Ref:           fmt.Sprintf("refs/heads/%s", config.Branch),
                    After:         latestCommit,
                    RepoURL:       fmt.Sprintf("https://github.com/%s/%s", config.Owner, config.Repo),
                    CommitMessage: fmt.Sprintf("Commit %s detected by github_pull_trigger", latestCommit),
                }

                err = triggerEventListener(config.EventListenerURL, payload)
                if err != nil {
                    fmt.Printf("Error triggering event listener: %v\n", err)
                } else {
                    err = storeCommit(latestCommit)
                    if err != nil {
                        fmt.Printf("Error storing new commit: %v\n", err)
                    } else {
                        fmt.Printf("Successfully triggered event and updated commit to %s\n", latestCommit)
                    }
                }
            }
        }
    }
}
