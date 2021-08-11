package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/andygrunwald/go-jira"
	"github.com/trivago/tgo/tcontainer"
	"github.com/xanzy/go-gitlab"
)

// cfgFile is globally set by the argument parser
var cfgFile string

var version string
var commit string

const JiraIssueDescriptionTemplate string = `
*_For newcomers: Before you log ANY time on this ticket, change the EPIC to your Onboarding epic!_*

h3. Story

"As <>, I want to <>, so I <>."

h3. Full description

Original [description from GitLab|{{ .GitlabURL }}]:

{{ .Description }}
 
h3. Completion criteria
 * Open a pull request to address any issues if applicable
 * Close the related GitLab [issue|{{ .GitlabURL }}]

h3. Documentation updates & improvements criteria
 * Up to the assignee

h3. Review timeline
 * PR to be sent for review by first Friday
 * First PR review to be completed by second Monday
`

// rootCmd is the main command used by the tool
var rootCmd = &cobra.Command{
	Use:   "jira-sync",
	Short: "Sync GitLab issue to Jira",
	Long:  `
Sync GitLab issues to Jira and marked the synced GitLab issues with a
"ticket-created" label.
`,
	Run: func(cmd *cobra.Command, args []string) {
		showVersion, err := cmd.Flags().GetBool("version")
		cobra.CheckErr(err)

		if showVersion {
			fmt.Printf("jira-sync version %s, commit %s\n", version, commit[:8])
			os.Exit(0)
		}

		jiraProject, err := cmd.Flags().GetString("jira-project")
		cobra.CheckErr(err)

		jiraEpic, err := cmd.Flags().GetString("jira-epic")
		cobra.CheckErr(err)

		jiraAccountName, err := cmd.Flags().GetString("jira-account")
		cobra.CheckErr(err)

		gitlabProject, err := cmd.Flags().GetString("gitlab-project")
		cobra.CheckErr(err)

		issueIDs, err := cmd.Flags().GetIntSlice("gitlab-issue")
		cobra.CheckErr(err)

		linkAsDescription, err := cmd.Flags().GetBool("link-as-description")
		cobra.CheckErr(err)

		token := viper.GetString("gitlab_token")
		if token == "" {
			cobra.CheckErr("no tokens were set in config")
		}

		jiraURL := viper.GetString("jira_url")
		if token == "" {
			cobra.CheckErr("no Jira URLs were set in config")
		}

		gitlabClient, err := gitlab.NewClient(token)
		cobra.CheckErr(err)

		jiraUsername := viper.GetString("jira_username")
		jiraTransport := jira.BasicAuthTransport{
			Username: jiraUsername,
			Password: viper.GetString("jira_password"),
		}

		jiraClient, err := jira.NewClient(jiraTransport.Client(), jiraURL)
		cobra.CheckErr(err)

		req, err := jiraClient.NewRequest("GET", "/rest/tempo-accounts/1/account/key/" + jiraAccountName, nil)
		cobra.CheckErr(err)

		var jiraAccount struct{
			ID int `json:"id"`
		}

		_, err = jiraClient.Do(req, &jiraAccount)
		cobra.CheckErr(err)

		gitlabIssues, err := getGitlabIssues(gitlabClient, gitlabProject, issueIDs)
		cobra.CheckErr(err)

		for _, gitlabIssue := range gitlabIssues {
			jiraIssue, err := syncToJira(jiraClient, jiraProject, jiraEpic, jiraAccount.ID, gitlabIssue, linkAsDescription)
			cobra.CheckErr(err)

			fmt.Printf("Jira issue %s is created from %d\n", jiraIssue.Key, gitlabIssue.IID)

			_, _, err = gitlabClient.Issues.UpdateIssue(gitlabProject, gitlabIssue.IID, &gitlab.UpdateIssueOptions{
				AddLabels: []string{"ticket created"},
			})
			cobra.CheckErr(err)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jira-sync.yaml)")

	rootCmd.Flags().StringP("gitlab-project", "", "", "GitLab project name")
	rootCmd.Flags().IntSliceP("gitlab-issue", "i", []int{}, "GitLab issue IDs")

	rootCmd.Flags().StringP("jira-project", "", "", "Jira project key (ex: SE)")
	rootCmd.Flags().StringP("jira-epic", "", "", "Jira epic key (ex: SE-1234)")
	rootCmd.Flags().StringP("jira-account", "", "", "Jira account key (ex: ABC)")

	rootCmd.Flags().BoolP("link-as-description", "", false, "use the GitLab issue link as the description of the jira ticket")
	rootCmd.Flags().BoolP("version", "", false, "show command version")

	var err error

	askVersion, err := rootCmd.Flags().GetBool("version")
	cobra.CheckErr(err)

	if askVersion {
		err = rootCmd.MarkFlagRequired("gitlab-project")
		cobra.CheckErr(err)

		err = rootCmd.MarkFlagRequired("gitlab-issue")
		cobra.CheckErr(err)

		err = rootCmd.MarkFlagRequired("jira-project")
		cobra.CheckErr(err)

		err = rootCmd.MarkFlagRequired("jira-epic")
		cobra.CheckErr(err)

		err = rootCmd.MarkFlagRequired("jira-account")
		cobra.CheckErr(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".jira-sync")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if _, err := fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed()); err != nil {
			cobra.CheckErr(err)
		}
	}
}

// Execute called by main program and executes the root cmd
func Execute(cmdVersion string, cmdCommit string) {
	version = cmdVersion
	commit = cmdCommit

	cobra.CheckErr(rootCmd.Execute())
}

func getGitlabIssues(client *gitlab.Client, projectID string, issueIDs []int) ([]*gitlab.Issue, error) {
	var issues []*gitlab.Issue

	for _, issueID := range issueIDs {
		issue, _, err := client.Issues.GetIssue(projectID, issueID, nil)
		if err != nil {
			return nil, err
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

func syncToJira(client *jira.Client, project string, epic string, accountID int, gitlabIssue *gitlab.Issue, linkAsDescription bool) (*jira.Issue, error) {
	var description string

	if !linkAsDescription {
		var descBuff bytes.Buffer
		descriptionTemplate := template.Must(template.New("description").Parse(JiraIssueDescriptionTemplate))
		descriptionTemplate.Execute(&descBuff, map[string]string{
			"Description": gitlabIssue.Description,
			"GitlabURL": gitlabIssue.WebURL,
		})

		description = descBuff.String()
	} else {
		description = gitlabIssue.WebURL
	}

	issue, _, err := client.Issue.Create(&jira.Issue{
		Fields: &jira.IssueFields{
			Type:        jira.IssueType{Name: "Story"},
			Project:     jira.Project{Key: project},
			Summary:     gitlabIssue.Title,
			Description: description,
			Unknowns: tcontainer.MarshalMap{
				"customfield_10006": epic,
				"customfield_10011": strconv.Itoa(accountID),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return issue, nil
}
